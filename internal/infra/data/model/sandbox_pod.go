package model

import "time"

// One sandbox per project; controller-owned lifecycle.
type SandboxPod struct {
	ID           string     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	ProjectID    string     `gorm:"column:project_id;not null;uniqueIndex"`
	UserID       int64      `gorm:"column:user_id;not null;index"`
	PodName      string     `gorm:"column:pod_name;not null"`
	Status       string     `gorm:"column:status;not null;default:'creating';index"`
	ErrorMessage *string    `gorm:"column:error_message"`
	StartedAt    *time.Time `gorm:"column:started_at"`
	StoppedAt    *time.Time `gorm:"column:stopped_at;index"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (SandboxPod) TableName() string { return "sandbox_pods" }
