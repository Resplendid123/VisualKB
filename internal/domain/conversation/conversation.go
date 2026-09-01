package conversation

import (
	"context"
	"learn/internal/domain"
	"strings"
	"time"
)

type Conversation struct {
	ID                string
	UserID            int64
	Title             string
	LastCompressionAt *time.Time
	ActiveProjectID   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ArchivedAt        *time.Time
}

type ConvoRepo interface {
	Create(ctx context.Context, convo *Conversation) error
	FindByIDAndUserID(ctx context.Context, id string, userID int64) (*Conversation, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]*Conversation, int64, error)
	UpdateActiveProject(ctx context.Context, convoID string, projectID *string) error
	Archive(ctx context.Context, id string, userID int64) error
}

func (c Conversation) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return domain.ErrInvalidConvoTitle
	}
	return nil
}
