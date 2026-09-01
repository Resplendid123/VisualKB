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

type projectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) project.ProjectRepo {
	return &projectRepo{db: db}
}

func (r *projectRepo) Create(ctx context.Context, p *project.Project) error {
	m := projectToModel(p)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	p.ID = m.ID
	return nil
}

func (r *projectRepo) FindByID(ctx context.Context, id string) (*project.Project, error) {
	var m model.Project
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return projectToDomain(&m), nil
}

func (r *projectRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*project.Project, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var ms []model.Project
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).Offset(offset).
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*project.Project, 0, len(ms))
	for i := range ms {
		out = append(out, projectToDomain(&ms[i]))
	}
	return out, nil
}

func (r *projectRepo) UpdateStatus(ctx context.Context, id, status string, errMsg *string) error {
	updates := map[string]any{"status": status}
	_ = errMsg
	return r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *projectRepo) UpdateTitle(ctx context.Context, id, title string) error {
	return r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", id).
		Update("title", title).Error
}

func (r *projectRepo) UpdateName(ctx context.Context, id, name string) error {
	return r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", id).
		Update("name", name).Error
}

func (r *projectRepo) UpdatePreviewURL(ctx context.Context, id, previewURL string) error {
	return r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", id).
		Update("preview_url", previewURL).Error
}

func (r *projectRepo) Archive(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.Project{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      project.ProjectStatusArchived,
			"archived_at": now,
		})
	return res.Error
}

func projectToDomain(m *model.Project) *project.Project {
	return &project.Project{
		ID:                   m.ID,
		UserID:               m.UserID,
		Name:                 m.Name,
		Title:                m.Title,
		Status:               m.Status,
		CreatedFromMessageID: m.CreatedFromMessageID,
		PreviewURL:           m.PreviewURL,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
		ArchivedAt:           m.ArchivedAt,
	}
}

func projectToModel(p *project.Project) *model.Project {
	return &model.Project{
		ID:                   p.ID,
		UserID:               p.UserID,
		Name:                 p.Name,
		Title:                p.Title,
		Status:               p.Status,
		CreatedFromMessageID: p.CreatedFromMessageID,
		PreviewURL:           p.PreviewURL,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
		ArchivedAt:           p.ArchivedAt,
	}
}

var _ project.ProjectRepo = (*projectRepo)(nil)
