package repo

import (
	"context"
	"errors"
	"time"

	"learn/internal/domain"
	"learn/internal/domain/project"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type sandboxRepo struct {
	db *gorm.DB
}

func NewSandboxRepo(db *gorm.DB) project.SandboxRepo {
	return &sandboxRepo{db: db}
}

func (r *sandboxRepo) Create(ctx context.Context, p *project.SandboxPod) error {
	m := sandboxPodToModel(p)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}

	p.ID = m.ID
	return nil
}

func (r *sandboxRepo) FindByProjectID(ctx context.Context, projectID string) (*project.SandboxPod, error) {
	var m model.SandboxPod
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSandboxNotFound
		}
		return nil, err
	}
	return sandboxPodToDomain(&m), nil
}

func (r *sandboxRepo) FindByID(ctx context.Context, id string) (*project.SandboxPod, error) {
	var m model.SandboxPod
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSandboxNotFound
		}
		return nil, err
	}
	return sandboxPodToDomain(&m), nil
}

func (r *sandboxRepo) UpdateStatus(ctx context.Context, id, status string, errMsg *string) error {
	res := r.db.WithContext(ctx).Model(&model.SandboxPod{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"error_message": errMsg,
		})
	return res.Error
}

func (r *sandboxRepo) MarkStarted(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.SandboxPod{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        project.SandboxStatusRunning,
			"started_at":    now,
			"error_message": nil,
		})
	return res.Error
}

func (r *sandboxRepo) MarkStopped(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.SandboxPod{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     project.SandboxStatusStopped,
			"stopped_at": now,
		})
	return res.Error
}

func sandboxPodToDomain(m *model.SandboxPod) *project.SandboxPod {
	return &project.SandboxPod{
		ID:           m.ID,
		ProjectID:    m.ProjectID,
		UserID:       m.UserID,
		PodName:      m.PodName,
		Status:       m.Status,
		ErrorMessage: m.ErrorMessage,
		StartedAt:    m.StartedAt,
		StoppedAt:    m.StoppedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func sandboxPodToModel(p *project.SandboxPod) *model.SandboxPod {
	return &model.SandboxPod{
		ID:           p.ID,
		ProjectID:    p.ProjectID,
		UserID:       p.UserID,
		PodName:      p.PodName,
		Status:       p.Status,
		ErrorMessage: p.ErrorMessage,
		StartedAt:    p.StartedAt,
		StoppedAt:    p.StoppedAt,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

var _ project.SandboxRepo = (*sandboxRepo)(nil)
