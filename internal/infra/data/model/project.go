package model

import "time"

type Project struct {
	ID                   string  `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID               int64   `gorm:"column:user_id;not null;index"`
	Name                 string  `gorm:"column:name;not null;size:64;index"`
	Title                string  `gorm:"column:title;not null;default:''"`
	Status               string  `gorm:"column:status;not null;default:'creating';index"`
	CreatedFromMessageID *string `gorm:"column:created_from_message_id;type:uuid;index"`
	// Controller-stamped CDN URL; "" until reconcile completes.
	PreviewURL string     `gorm:"column:preview_url;not null;default:''"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	ArchivedAt *time.Time `gorm:"column:archived_at;index"`
}

func (Project) TableName() string { return "projects" }
