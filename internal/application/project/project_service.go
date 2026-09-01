package project

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"learn/internal/domain"
	domainconv "learn/internal/domain/conversation"
	domainproject "learn/internal/domain/project"
)

// ProjectService binds project to sandbox.
type ProjectService struct {
	projectRepo domainproject.ProjectRepo
	convoRepo   domainconv.ConvoRepo
	sandbox     domainproject.Runtime
}

func NewProjectService(
	projectRepo domainproject.ProjectRepo,
	convoRepo domainconv.ConvoRepo,
	sandbox domainproject.Runtime,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		convoRepo:   convoRepo,
		sandbox:     sandbox,
	}
}

func (s *ProjectService) CreateFromChat(
	ctx context.Context,
	userID int64,
	convoID, messageID, name string,
) (*domainproject.Project, error) {
	name = strings.TrimSpace(name)
	if err := domainproject.ValidateProjectName(name); err != nil {
		return nil, err
	}

	var msgPtr *string
	if messageID != "" {
		msgPtr = &messageID
	}
	p := &domainproject.Project{
		UserID:               userID,
		Name:                 name,
		Title:                name,
		Status:               domainproject.ProjectStatusReady,
		CreatedFromMessageID: msgPtr,
	}
	if err := s.projectRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	pid := p.ID
	if err := s.convoRepo.UpdateActiveProject(ctx, convoID, &pid); err != nil {
		return nil, err
	}
	// Sandbox start failure does not block.
	if err := s.sandbox.EnsureRunning(ctx, userID, pid); err != nil {
		slog.WarnContext(ctx, "ensure sandbox on project create failed",
			"project", pid, "user", userID, "err", err)
	}
	p, err := s.projectRepo.FindByID(ctx, pid)
	if err != nil {
		return nil, err
	}
	s.stampPreviewURL(ctx, userID, p)
	return p, nil
}

// List returns non-archived projects, marks conversation active.
func (s *ProjectService) List(ctx context.Context, userID int64, convoID string, limit, offset int) ([]*domainproject.Project, error) {
	ps, err := s.projectRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	var activeID string
	if convoID != "" {

		convo, err := s.convoRepo.FindByIDAndUserID(ctx, convoID, userID)
		if err == nil && convo != nil && convo.ActiveProjectID != nil {
			activeID = *convo.ActiveProjectID
		}
	}
	out := make([]*domainproject.Project, 0, len(ps))
	for _, p := range ps {
		if p.IsAvailable() {
			p.IsActive = p.ID == activeID
			out = append(out, p)
		}
	}
	return out, nil
}

// Get fetches project by ID.
func (s *ProjectService) Get(ctx context.Context, userID int64, id string) (*domainproject.Project, error) {
	p, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, domain.ErrProjectNotFound
	}
	return p, nil
}

func (s *ProjectService) GetActive(ctx context.Context, userID int64, convoID string) (*domainproject.Project, error) {
	convo, err := s.convoRepo.FindByIDAndUserID(ctx, convoID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrConvoNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if convo.ActiveProjectID == nil {
		return nil, nil
	}
	p, err := s.projectRepo.FindByID(ctx, *convo.ActiveProjectID)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if !p.IsAvailable() {
		return nil, nil
	}
	s.stampPreviewURL(ctx, userID, p)
	return p, nil
}

// SetActive binds project or clears binding.
func (s *ProjectService) SetActive(ctx context.Context, userID int64, convoID, projectID string) (*domainproject.Project, error) {
	if _, err := s.convoRepo.FindByIDAndUserID(ctx, convoID, userID); err != nil {
		return nil, err
	}
	if projectID == "" {
		// Empty projectID writes NULL (UUID column).
		if err := s.convoRepo.UpdateActiveProject(ctx, convoID, nil); err != nil {
			return nil, err
		}
		return nil, nil
	}
	p, err := s.Get(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !p.IsAvailable() {
		return nil, domain.ErrProjectNotFound
	}
	if err := s.convoRepo.UpdateActiveProject(ctx, convoID, &projectID); err != nil {
		return nil, err
	}
	s.stampPreviewURL(ctx, userID, p)
	return p, nil
}

// Archive marks archived and suspends sandbox.
func (s *ProjectService) Archive(ctx context.Context, userID int64, id string) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := p.Archive(); err != nil {
		return err
	}
	if err := s.projectRepo.Archive(ctx, p.ID); err != nil {
		return err
	}
	// Archive drops CR; bucket + ns persist.
	if err := s.sandbox.DeleteSandbox(ctx, userID, id); err != nil {
		slog.WarnContext(ctx, "delete sandbox on project archive failed",
			"project", id, "user", userID, "err", err)
	}
	return nil
}

// CreateForUser creates user-initiated sidebar project.
func (s *ProjectService) CreateForUser(ctx context.Context, userID int64, title string) (*domainproject.Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "未命名"
	}
	p := &domainproject.Project{
		UserID: userID,
		Name:   "user-pending",
		Title:  title,
		Status: domainproject.ProjectStatusReady,
	}
	if err := s.projectRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	// Placeholder name uses full ID, never truncated.
	if err := s.projectRepo.UpdateName(ctx, p.ID, "user-"+p.ID); err != nil {
		return nil, err
	}
	if err := s.sandbox.EnsureRunning(ctx, userID, p.ID); err != nil {
		slog.WarnContext(ctx, "ensure sandbox on project create failed",
			"project", p.ID, "user", userID, "err", err)
	}
	p, err := s.projectRepo.FindByID(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	s.stampPreviewURL(ctx, userID, p)
	return p, nil
}

// Rename updates Title only, preserves slug.
func (s *ProjectService) Rename(ctx context.Context, userID int64, id, title string) (*domainproject.Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrInvalidProjectName
	}
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !p.IsAvailable() {
		return nil, domain.ErrProjectNotFound
	}
	if err := s.projectRepo.UpdateTitle(ctx, id, title); err != nil {
		return nil, err
	}
	p.Title = title
	return p, nil
}

// stampPreviewURL writes controller-stamped URL back to DB once.
// S3+bucket are the source of truth; controller CR is durable.
func (s *ProjectService) stampPreviewURL(ctx context.Context, userID int64, p *domainproject.Project) {
	if p.PreviewURL != "" {
		return
	}
	sb, err := s.sandbox.GetStatus(ctx, userID, p.ID)
	if err != nil || sb == nil || sb.PublicURL == "" {
		return
	}
	if err := s.projectRepo.UpdatePreviewURL(ctx, p.ID, sb.PublicURL); err != nil {
		slog.WarnContext(ctx, "persist preview_url failed",
			"project", p.ID, "user", userID, "err", err)
		return
	}
	p.PreviewURL = sb.PublicURL
}

var _ domainproject.ProjectOps = (*ProjectService)(nil)
