package project

import (
	"context"
	"time"
)

type Command struct {
	Raw     string
	Timeout time.Duration
}

// Project-scoped; conversations share one sandbox.
type Runtime interface {
	EnsureRunning(ctx context.Context, userID int64, projectID string) error

	Exec(ctx context.Context, userID int64, projectID, command string, timeout time.Duration) (string, error)

	// Snapshot read of CR-stamped status.
	GetStatus(ctx context.Context, userID int64, projectID string) (*SandboxStatus, error)

	DeleteSandbox(ctx context.Context, userID int64, projectID string) error
}
