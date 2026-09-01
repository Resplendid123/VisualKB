package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"learn/internal/domain"
	domainproject "learn/internal/domain/project"
	"learn/internal/infra/config"
	sandboxpkg "learn/internal/infra/sandbox"
)

// SandboxService wraps controller sandbox API.
type SandboxService struct {
	sandboxRepo     domainproject.SandboxRepo
	execRepo        domainproject.ExecutionRepo
	sandboxMgr      domainproject.SandboxManager
	runner          domainproject.CommandRunner
	podReadyTimeout time.Duration
}

func NewSandboxService(
	sandboxRepo domainproject.SandboxRepo,
	execRepo domainproject.ExecutionRepo,
	sandboxMgr domainproject.SandboxManager,
	runner domainproject.CommandRunner,
	cfg config.SandboxConfig,
) *SandboxService {
	return &SandboxService{
		sandboxRepo:     sandboxRepo,
		execRepo:        execRepo,
		sandboxMgr:      sandboxMgr,
		runner:          runner,
		podReadyTimeout: cfg.PodReadyTimeout,
	}
}

// EnsureRunning implements domainproject.Runtime.
func (s *SandboxService) EnsureRunning(ctx context.Context, userID int64, projectID string) error {
	return s.start(ctx, userID, projectID)
}

// Exec implements domainproject.Runtime.
func (s *SandboxService) Exec(ctx context.Context, userID int64, projectID, command string, timeout time.Duration) (string, error) {
	row, err := s.sandboxRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("sandbox for project %s: %w", projectID, err)
	}
	return s.runner.Exec(ctx, row.PodName, domainproject.UserNamespace(userID), command, timeout)
}

// GetStatus implements domainproject.Runtime; passes through to manager.
func (s *SandboxService) GetStatus(ctx context.Context, userID int64, projectID string) (*domainproject.SandboxStatus, error) {
	return s.sandboxMgr.GetStatus(ctx, userID, projectID)
}

// DeleteSandbox removes CR, keeps bucket.
func (s *SandboxService) DeleteSandbox(ctx context.Context, userID int64, projectID string) error {
	row, err := s.sandboxRepo.FindByProjectID(ctx, projectID)
	if err != nil && !errors.Is(err, domain.ErrSandboxNotFound) {
		return err
	}
	if err := s.sandboxMgr.DeleteSandbox(ctx, userID, projectID); err != nil {
		return fmt.Errorf("delete sandbox: %w", err)
	}
	if row != nil {
		if err := s.sandboxRepo.MarkStopped(ctx, row.ID); err != nil {
			slog.WarnContext(ctx, "mark sandbox stopped on delete failed", "err", err, "id", row.ID)
		}
	}
	return nil
}

// ExecCommand runs command in project sandbox.
func (s *SandboxService) ExecCommand(
	ctx context.Context,
	userID int64,
	projectID, convoID, messageID, command string,
	timeout time.Duration,
) (string, error) {
	if command == "" {
		return "", fmt.Errorf("%w: command is required", domain.ErrExecDenied)
	}
	if projectID == "" {
		// Tell LLM to bootstrap via create_project(name).
		return "", domain.ErrNoActiveProject
	}

	if denied, reason := checkDenylist(command); denied {
		row := &domainproject.SandboxExecution{
			UserID:         userID,
			ProjectID:      projectID,
			ConversationID: convoID,
			Command:        command,
			Status:         domainproject.ExecDenied,
			OutputTail:     &reason,
		}
		_ = s.execRepo.Create(ctx, row)
		return "", fmt.Errorf("%w: %s", domain.ErrExecDenied, reason)
	}

	if err := s.EnsureRunning(ctx, userID, projectID); err != nil {
		return "", err
	}

	row, err := s.sandboxRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return "", err
	}

	execRow := &domainproject.SandboxExecution{
		SandboxPodID:   row.ID,
		UserID:         userID,
		ProjectID:      projectID,
		ConversationID: convoID,
		Command:        command,
		Status:         domainproject.ExecRunning,
	}
	if messageID != "" {
		mid := messageID
		execRow.MessageID = &mid
	}
	if err := s.execRepo.Create(ctx, execRow); err != nil {
		slog.WarnContext(ctx, "exec audit create failed", "err", err, "user", userID)
	}

	// Note: controller updates lastActivity on exec path.

	started := time.Now()
	content, execErr := s.runner.Exec(ctx, row.PodName, domainproject.UserNamespace(row.UserID), command, timeout)
	durationMs := time.Since(started).Milliseconds()

	status := domainproject.ExecSuccess
	if execErr != nil {
		if errors.Is(execErr, domain.ErrExecTimeout) {
			status = domainproject.ExecTimeout
		} else {
			status = domainproject.ExecFailed
		}
	}
	var outputTailPtr *string
	if content != "" {
		t := tailStr(content, 16*1024)
		outputTailPtr = &t
	}
	if markErr := s.execRepo.MarkFinished(ctx, execRow.ID, status, nil, durationMs, outputTailPtr); markErr != nil {
		slog.WarnContext(ctx, "exec audit finish failed", "err", markErr, "exec_id", execRow.ID)
	}

	return content, execErr
}

// start creates or resumes CR, waits Ready.
func (s *SandboxService) start(ctx context.Context, userID int64, projectID string) error {
	podName := sandboxpkg.SandboxName(projectID)
	prev, err := s.sandboxRepo.FindByProjectID(ctx, projectID)
	if err != nil && !errors.Is(err, domain.ErrSandboxNotFound) {
		return err
	}

	rowID := ""
	if prev == nil {
		row := &domainproject.SandboxPod{
			ProjectID: projectID,
			UserID:    userID,
			PodName:   podName,
			Status:    domainproject.SandboxStatusCreating,
		}
		if err := s.sandboxRepo.Create(ctx, row); err != nil {
			if errors.Is(err, domain.ErrSandboxAlreadyUp) {
				return nil
			}
			return domain.ErrSandboxCreateFail
		}
		rowID = row.ID
	} else {
		rowID = prev.ID
	}

	spec := defaultRuntimeSpec()
	if err := s.sandboxMgr.ApplySandbox(ctx, userID, projectID, spec); err != nil {
		msg := err.Error()
		if uerr := s.sandboxRepo.UpdateStatus(ctx, rowID, domainproject.SandboxStatusErrored, &msg); uerr != nil {
			slog.WarnContext(ctx, "update status to errored failed", "err", uerr, "pod", podName)
		}
		return fmt.Errorf("apply sandbox %s: %w", podName, err)
	}
	if err := s.sandboxMgr.WaitRunning(ctx, userID, projectID, s.podReadyTimeout); err != nil {
		msg := err.Error()
		if uerr := s.sandboxRepo.UpdateStatus(ctx, rowID, domainproject.SandboxStatusErrored, &msg); uerr != nil {
			slog.WarnContext(ctx, "update status to errored failed", "err", uerr, "pod", podName)
		}
		return fmt.Errorf("wait sandbox running %s: %w", podName, err)
	}

	if err := s.sandboxRepo.MarkStarted(ctx, rowID); err != nil {
		return err
	}
	return nil
}

// defaultRuntimeSpec builds long-running sandbox image.
func defaultRuntimeSpec() any {
	return sandboxpkg.AgentSandboxSpec{
		Runtime: sandboxpkg.RuntimeSection{
			Image: "node:lts-bookworm-slim",
			Cmd:   []string{"sleep", "infinity"},
		},
	}
}

// checkDenylist applies coarse string blacklist.
func checkDenylist(cmd string) (bool, string) {
	low := strings.ToLower(cmd)
	checks := []struct {
		needle string
		reason string
	}{
		{"rm -rf /workspace/documents", "documents/ is read-only"},
		{"rm -rf /workspace/notes", "notes/ is read-only"},
		{"mkfs", "filesystem format denied"},
		{"dd if=/dev/zero", "raw disk write denied"},
	}
	for _, c := range checks {
		if strings.Contains(low, c.needle) {
			return true, c.reason
		}
	}
	return false, ""
}

func tailStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "...(truncated)\n" + s[len(s)-max:]
}

var (
	_ domainproject.Runtime         = (*SandboxService)(nil)
	_ domainproject.CommandExecutor = (*SandboxService)(nil)
)
