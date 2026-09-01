package model

import "time"

type Document struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;column:id"`
	CurrentVersionID *int64     `gorm:"column:current_version_id;index"`
	UserID           int64      `gorm:"column:user_id;not null;index:idx_user_source,priority:1"`
	Source           string     `gorm:"column:source;not null;check:source IN ('note','knowledge');index:idx_user_source,priority:2"`
	Title            string     `gorm:"column:title;not null"`
	Lang             string     `gorm:"column:lang;default:zh"`
	ContentType      string     `gorm:"column:content_type;not null;default:'markdown'"`
	ChunkStatus      int8       `gorm:"column:chunk_status;not null;default:0"`
	ArchivedAt       *time.Time `gorm:"column:archived_at;index"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt        *time.Time `gorm:"column:updated_at"`
}

func (Document) TableName() string { return "documents" }
