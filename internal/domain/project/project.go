package project

import (
	"context"
	"learn/internal/domain"
	"regexp"
	"strconv"
	"time"
)

var ProjectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

const (
	ProjectStatusCreating = "creating"
	ProjectStatusReady    = "ready"
	ProjectStatusArchived = "archived"
)

type Project struct {
	ID                   string
	UserID               int64
	Name                 string
	Title                string
	Status               string
	CreatedFromMessageID *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ArchivedAt           *time.Time
	IsActive             bool
	PreviewURL           string // controller-stamped CDN URL ("" until ready)
}

func (p *Project) IsAvailable() bool {
	return p.Status == ProjectStatusReady
}

func (p *Project) Archive() error {
	if p.Status == ProjectStatusArchived {
		return nil
	}
	p.Status = ProjectStatusArchived
	return nil
}

func (p *Project) ContainerPath() string {
	return "/workspace/project"
}

func (p *Project) HostPath() string {
	return "/srv/jfs/projects/" + strconv.FormatInt(p.UserID, 10) + "/" + p.Name
}

type ProjectRepo interface {
	Create(ctx context.Context, p *Project) error
	FindByID(ctx context.Context, id string) (*Project, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*Project, error)
	UpdateStatus(ctx context.Context, id, status string, errMsg *string) error
	UpdateTitle(ctx context.Context, id, title string) error
	UpdateName(ctx context.Context, id, name string) error
	UpdatePreviewURL(ctx context.Context, id, previewURL string) error
	Archive(ctx context.Context, id string) error
}

// Port for create_project and list_projects tools.
type ProjectOps interface {
	CreateFromChat(ctx context.Context, userID int64, convoID, messageID, name string) (*Project, error)
	List(ctx context.Context, userID int64, convoID string, limit, offset int) ([]*Project, error)
}

func ValidateProjectName(name string) error {
	if name == "" {
		return domain.ErrInvalidProjectName
	}
	if !ProjectNamePattern.MatchString(name) {
		return domain.ErrInvalidProjectName
	}
	return nil
}
