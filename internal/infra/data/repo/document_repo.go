package repo

import (
	"context"
	"errors"

	"learn/internal/domain"
	"learn/internal/domain/document"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type documentRepo struct {
	db *gorm.DB
}

func NewDocumentRepo(db *gorm.DB) document.DocumentRepo {
	return &documentRepo{db: db}
}

func (r *documentRepo) FindByID(ctx context.Context, id int64) (*document.Document, error) {
	var m model.Document
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}
	return documentToDomain(&m), nil
}

func (r *documentRepo) Create(ctx context.Context, doc *document.Document) error {
	if doc.ID != 0 {
		return errors.New("create: document id must be zero (DB-generated)")
	}
	if doc.ChunkStatus != document.ChunkStatusDirty {
		return errors.New("create: chunk_status must default to Dirty (0)")
	}
	m := model.Document{
		UserID:      doc.UserID,
		Source:      doc.Source,
		Title:       doc.Title,
		Lang:        doc.Lang,
		ContentType: string(doc.ContentType),
		ChunkStatus: int8(doc.ChunkStatus),
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	doc.ID = m.ID
	doc.CreatedAt = m.CreatedAt
	return nil
}

func (r *documentRepo) UpdateCurrentVersion(ctx context.Context, documentID int64, versionID int64) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", documentID).
		Update("current_version_id", versionID).Error
}

func (r *documentRepo) UpdateTitleAndLang(ctx context.Context, documentID int64, title, lang string) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", documentID).
		Updates(map[string]any{
			"title":      title,
			"lang":       lang,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *documentRepo) MarkChunked(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", id).
		Update("chunk_status", document.ChunkStatusChunked).Error
}

func (r *documentRepo) MarkError(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", id).
		Update("chunk_status", document.ChunkStatusError).Error
}

func (r *documentRepo) ListDirty(ctx context.Context, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var ids []int64
	err := r.db.WithContext(ctx).Model(&model.Document{}).
		Where("chunk_status = ?", document.ChunkStatusDirty).
		Where("archived_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *documentRepo) ListUnchunkedByUser(ctx context.Context, userID int64, source string) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&model.Document{}).
		Where("user_id = ? AND source = ? AND chunk_status <> ?", userID, source, document.ChunkStatusChunked).
		Where("archived_at IS NULL").
		Order("id ASC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *documentRepo) FindChunkStatus(ctx context.Context, id int64) (document.ChunkStatus, error) {
	var status int8
	err := r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", id).
		Pluck("chunk_status", &status).Error
	if err != nil {
		return 0, err
	}
	return document.ChunkStatus(status), nil
}

func (r *documentRepo) ListByUser(
	ctx context.Context,
	userID int64,
	opts document.ListOpts,
) ([]*document.Document, int64, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	q := r.db.WithContext(ctx).Model(&model.Document{}).Where("user_id = ?", userID)
	if !opts.IncludeArchived {
		q = q.Where("archived_at IS NULL")
	}
	if opts.Source != "" {
		q = q.Where("source = ?", opts.Source)
	}
	if opts.ChunkStatus != nil {
		q = q.Where("chunk_status = ?", int8(*opts.ChunkStatus))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Document
	if err := q.Order("id DESC").
		Limit(limit).
		Offset(opts.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*document.Document, len(rows))
	for i := range rows {
		out[i] = documentToDomain(&rows[i])
	}
	return out, total, nil
}

func (r *documentRepo) Archive(ctx context.Context, documentID, userID int64) error {
	res := r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ? AND user_id = ? AND archived_at IS NULL", documentID, userID).
		Update("archived_at", gorm.Expr("NOW()"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *documentRepo) MarkDirty(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", id).
		Update("chunk_status", document.ChunkStatusDirty).Error
}

var _ document.DocumentRepo = (*documentRepo)(nil)

func documentToDomain(m *model.Document) *document.Document {
	return &document.Document{
		ID:               m.ID,
		CurrentVersionID: m.CurrentVersionID,
		UserID:           m.UserID,
		Source:           m.Source,
		Title:            m.Title,
		Lang:             m.Lang,
		ContentType:      document.ContentType(m.ContentType),
		ChunkStatus:      document.ChunkStatus(m.ChunkStatus),
		ArchivedAt:       m.ArchivedAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
