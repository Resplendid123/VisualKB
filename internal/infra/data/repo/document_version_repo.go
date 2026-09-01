package repo

import (
	"context"
	"errors"

	"learn/internal/domain"
	"learn/internal/domain/document"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type documentVersionRepo struct {
	db *gorm.DB
}

func NewDocumentVersionRepo(db *gorm.DB) document.DocumentVersionRepo {
	return &documentVersionRepo{db: db}
}

func (r *documentVersionRepo) FindCurrent(ctx context.Context, doc *document.Document) (*document.DocumentVersion, error) {
	if doc.CurrentVersionID == nil {
		return nil, domain.ErrDocumentVersionNotFound
	}
	var m model.DocumentVersion
	if err := r.db.WithContext(ctx).Where("id = ?", *doc.CurrentVersionID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentVersionNotFound
		}
		return nil, err
	}
	return versionToDomain(&m), nil
}

func (r *documentVersionRepo) NextVersionNumber(ctx context.Context, documentID int64) (int, error) {
	var maxV int
	err := r.db.WithContext(ctx).Model(&model.DocumentVersion{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxV).Error
	if err != nil {
		return 0, err
	}
	return maxV + 1, nil
}

func (r *documentVersionRepo) Create(ctx context.Context, ver *document.DocumentVersion) error {
	if ver.ID != 0 {
		return errors.New("create: document version id must be zero (DB-generated)")
	}
	m := model.DocumentVersion{
		DocumentID: ver.DocumentID,
		Version:    ver.Version,
		Title:      ver.Title,
		FileKey:    ver.FileKey,
		FileSize:   ver.FileSize,
		FileHash:   ver.FileHash,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	ver.ID = m.ID
	ver.CreatedAt = m.CreatedAt
	return nil
}

func (r *documentVersionRepo) UpdateFileKey(ctx context.Context, versionID int64, fileKey string) error {
	return r.db.WithContext(ctx).Model(&model.DocumentVersion{}).
		Where("id = ?", versionID).
		Updates(map[string]any{
			"file_key":   fileKey,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *documentVersionRepo) ListByDocument(ctx context.Context, documentID int64) ([]*document.DocumentVersion, error) {
	var rows []model.DocumentVersion
	if err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("version ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*document.DocumentVersion, len(rows))
	for i := range rows {
		out[i] = versionToDomain(&rows[i])
	}
	return out, nil
}

var _ document.DocumentVersionRepo = (*documentVersionRepo)(nil)

func versionToDomain(m *model.DocumentVersion) *document.DocumentVersion {
	return &document.DocumentVersion{
		ID:         m.ID,
		DocumentID: m.DocumentID,
		Version:    m.Version,
		Title:      m.Title,
		FileKey:    m.FileKey,
		FileSize:   m.FileSize,
		FileHash:   m.FileHash,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
