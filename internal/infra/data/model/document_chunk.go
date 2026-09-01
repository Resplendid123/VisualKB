package model

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type Chunk struct {
	ID         int64           `gorm:"primaryKey;autoIncrement;column:id"`
	ParentID   *int64          `gorm:"column:parent_id;index"`
	DocumentID int64           `gorm:"column:document_id;not null;index"`
	ChunkIndex int             `gorm:"column:chunk_index;not null"`
	Content    string          `gorm:"column:content;not null"`
	Header     *string         `gorm:"column:header"`
	TokenCount *int            `gorm:"column:token_count"`
	Embedding  pgvector.Vector `gorm:"column:embedding;type:vector(1024)"`
	CreatedAt  time.Time       `gorm:"column:created_at;not null;default:now()"`
}

func (Chunk) TableName() string { return "document_chunks" }
