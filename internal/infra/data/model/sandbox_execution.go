package model

import "time"

// One audit row per command execution.
type SandboxExecution struct {
	ID             string     `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	SandboxPodID   string     `gorm:"column:sandbox_pod_id;not null;index;type:uuid"`
	UserID         int64      `gorm:"column:user_id;not null;index"`
	ProjectID      string     `gorm:"column:project_id;type:uuid;index"`
	ConversationID string     `gorm:"column:conversation_id;type:uuid;index"`
	MessageID      *string    `gorm:"column:message_id;type:uuid;index"`
	Command        string     `gorm:"column:command;not null"`
	ExitCode       *int       `gorm:"column:exit_code"`
	DurationMs     *int64     `gorm:"column:duration_ms"`
	Status         string     `gorm:"column:status;not null;default:'running';index"`
	OutputTail     *string    `gorm:"column:output_tail;type:text"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;index"`
	FinishedAt     *time.Time `gorm:"column:finished_at"`
}

func (SandboxExecution) TableName() string { return "sandbox_executions" }
