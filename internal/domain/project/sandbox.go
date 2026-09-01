package project

import (
	"context"
	"strconv"
	"time"
)

const (
	SandboxStatusCreating = "creating"
	SandboxStatusRunning  = "running"
	SandboxStatusStopped  = "stopped"
	SandboxStatusErrored  = "errored"
)

// Audit row only; controller owns real phase.
type SandboxPod struct {
	ID           string
	ProjectID    string
	UserID       int64
	PodName      string
	Status       string
	ErrorMessage *string
	StartedAt    *time.Time
	StoppedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProjectID is the unique pod key.
type SandboxRepo interface {
	Create(ctx context.Context, p *SandboxPod) error
	FindByProjectID(ctx context.Context, projectID string) (*SandboxPod, error)
	FindByID(ctx context.Context, id string) (*SandboxPod, error)
	UpdateStatus(ctx context.Context, id, status string, errMsg *string) error
	MarkStarted(ctx context.Context, id string) error
	MarkStopped(ctx context.Context, id string) error
}

// Controller recreates this from spec.tenantID.
func UserNamespace(userID int64) string {
	return "sandbox-u-" + strconv.FormatInt(userID, 10)
}
