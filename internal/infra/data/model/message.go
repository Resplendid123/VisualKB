package model

import (
	"encoding/json"
	"time"
)

type Message struct {
	ID               string          `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	ConversationID   string          `gorm:"column:conversation_id;not null;type:uuid"`
	Role             string          `gorm:"column:role;not null"`
	Content          *string         `gorm:"column:content"`
	Seq              int64           `gorm:"column:seq;not null;default:0"`
	TurnID           int64           `gorm:"column:turn_id;not null;default:0"`
	SeqID            int64           `gorm:"column:seq_id;not null;default:0"`
	IsModified       bool            `gorm:"column:is_modified;not null;default:false"`
	IsCompressed     bool            `gorm:"column:is_compressed;not null;default:false"`
	IsToolCompressed bool            `gorm:"column:is_tool_compressed;not null;default:false"`
	ToolCalls        json.RawMessage `gorm:"column:tool_calls;type:jsonb;serializer:json"`
	ToolCallID       *string         `gorm:"column:tool_call_id;index"`
	ToolName         *string         `gorm:"column:tool_name"`
	PromptTokens     *int64          `gorm:"column:prompt_tokens"`
	CompletionTokens *int64          `gorm:"column:completion_tokens"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt        time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (Message) TableName() string { return "messages" }
