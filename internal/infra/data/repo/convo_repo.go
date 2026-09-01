package repo

import (
	"context"
	"errors"
	"time"

	"learn/internal/domain"
	"learn/internal/domain/conversation"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type convoRepo struct {
	db *gorm.DB
}

func NewConvoRepo(db *gorm.DB) conversation.ConvoRepo {
	return &convoRepo{db: db}
}

func (r *convoRepo) Create(ctx context.Context, convo *conversation.Conversation) error {
	m := convoToModel(convo)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	convo.ID = m.ID
	return nil
}

func (r *convoRepo) FindByIDAndUserID(ctx context.Context, id string, userID int64) (*conversation.Conversation, error) {
	var m model.Conversation
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", id, userID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrConvoNotFound
		}
		return nil, err
	}
	return convoToDomain(&m), nil
}

func (r *convoRepo) List(ctx context.Context, userID int64, limit, offset int) ([]*conversation.Conversation, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("user_id = ? AND archived_at IS NULL", userID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ms []model.Conversation
	if err := q.Order("updated_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*conversation.Conversation, 0, len(ms))
	for i := range ms {
		out = append(out, convoToDomain(&ms[i]))
	}
	return out, total, nil
}

func (r *convoRepo) UpdateActiveProject(ctx context.Context, convoID string, projectID *string) error {
	res := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ?", convoID).
		Update("active_project_id", projectID)
	return res.Error
}

func (r *convoRepo) Archive(ctx context.Context, id string, userID int64) error {
	res := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", id, userID).
		Update("archived_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrConvoNotFound
	}
	return nil
}

func convoToDomain(m *model.Conversation) *conversation.Conversation {
	return &conversation.Conversation{
		ID:                m.ID,
		UserID:            m.UserID,
		Title:             m.Title,
		LastCompressionAt: m.LastCompressionAt,
		ActiveProjectID:   m.ActiveProjectID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		ArchivedAt:        m.ArchivedAt,
	}
}

func convoToModel(c *conversation.Conversation) *model.Conversation {
	return &model.Conversation{
		ID:                c.ID,
		UserID:            c.UserID,
		Title:             c.Title,
		LastCompressionAt: c.LastCompressionAt,
		ActiveProjectID:   c.ActiveProjectID,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
		ArchivedAt:        c.ArchivedAt,
	}
}

var _ conversation.ConvoRepo = (*convoRepo)(nil)
