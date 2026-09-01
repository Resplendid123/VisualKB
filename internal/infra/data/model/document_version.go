package model

import "time"

type DocumentVersion struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id"`
	DocumentID int64      `gorm:"column:document_id;not null;index:idx_doc_version;constraint:OnDelete:CASCADE;"`
	Version    int        `gorm:"column:version;not null;default:1"`
	Title      string     `gorm:"column:title;not null"`
	FileKey    string     `gorm:"column:file_key;not null"`
	FileSize   int64      `gorm:"column:file_size;not null"`
	FileHash   *string    `gorm:"column:file_hash"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (DocumentVersion) TableName() string { return "document_versions" }
