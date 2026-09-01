package repo

import (
	"context"

	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type ChunkRepo interface {
	DeleteByDocumentID(ctx context.Context, documentID int64) error
	BatchInsert(ctx context.Context, chunks []model.Chunk) error
}

type documentChunkRepo struct {
	db *gorm.DB
}

func NewDocumentChunkRepo(db *gorm.DB) ChunkRepo {
	return &documentChunkRepo{db: db}
}

func (r *documentChunkRepo) DeleteByDocumentID(ctx context.Context, documentID int64) error {
	return r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Delete(&model.Chunk{}).Error
}

func (r *documentChunkRepo) BatchInsert(ctx context.Context, chunks []model.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&chunks).Error
	})
}
