package project

import (
	"context"
	"time"
)

const (
	ExecRunning = "running"
	ExecSuccess = "success"
	ExecFailed  = "failed"
	ExecTimeout = "timeout"
	ExecDenied  = "denied"
)

// ConversationID retained for audit tracing.
type SandboxExecution struct {
	ID             string
	SandboxPodID   string
	UserID         int64
	ProjectID      string
	ConversationID string
	MessageID      *string
	Command        string
	ExitCode       *int
	DurationMs     *int64
	Status         string
	OutputTail     *string
	CreatedAt      time.Time
	FinishedAt     *time.Time
}

type ExecutionRepo interface {
	Create(ctx context.Context, e *SandboxExecution) error
	FindByID(ctx context.Context, id string) (*SandboxExecution, error)
	MarkFinished(ctx context.Context, id, status string, exitCode *int, durationMs int64, outputTail *string) error
}

// Runs a command in an existing pod.
type CommandRunner interface {
	Exec(ctx context.Context, podName, namespace, command string, timeout time.Duration) (string, error)
}

// Backed by Agent-Sandbox-Controller; one CR per project.
type SandboxManager interface {
	ApplySandbox(ctx context.Context, userID int64, projectID string, spec any) error
	// Polls until phase=Running or terminal failure.
	WaitRunning(ctx context.Context, userID int64, projectID string, timeout time.Duration) error
	// Like WaitRunning but returns URL details.
	FetchRunning(ctx context.Context, userID int64, projectID string, timeout time.Duration) (*SandboxStatus, error)
	// Snapshot read; URL may be "" pre-reconcile.
	GetStatus(ctx context.Context, userID int64, projectID string) (*SandboxStatus, error)
	DeleteSandbox(ctx context.Context, userID int64, projectID string) error
}

// Controller-populated; new fields need controller support.
type SandboxStatus struct {
	Phase       string // "Pending" | "Running" | "Suspended" | "Expired" | "Failed"
	PublicURL   string // pre-allocated CDN URL, stamped each reconcile
	PreviewHost string // {tenant}-{project}.preview.example.com — controller-owned
	PodName     string // runtime pod owned by this sandbox
	Bucket      string // visualkb-{tenant}-proj-{project}
}

// Port used by the bash chat tool.
type CommandExecutor interface {
	// convoID is audit-only; projectID keys the sandbox.
	ExecCommand(ctx context.Context, userID int64, projectID, convoID, messageID, command string, timeout time.Duration) (string, error)
}
