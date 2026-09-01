package model

import "time"

type Conversation struct {
	ID                string     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID            int64      `gorm:"column:user_id;index"`
	Title             string     `gorm:"column:title;type:varchar(255)"`
	LastCompressionAt *time.Time `gorm:"column:last_compression_at"`
	ActiveProjectID   *string    `gorm:"column:active_project_id;type:uuid;index"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	ArchivedAt        *time.Time `gorm:"column:archived_at;index"`
}

func (Conversation) TableName() string { return "conversations" }
