package conversation

import (
	"context"
	"time"
)

const MessageCacheTTL = 5 * time.Minute

type Message struct {
	ID               string
	ConversationID   string
	Role             string
	Content          *string
	Seq              int64
	TurnID           int64
	SeqID            int64
	IsModified       bool
	IsCompressed     bool
	IsToolCompressed bool
	ToolCalls        []ToolCallData
	ToolCallID       *string
	ToolName         *string
	TokenUsage       *TokenUsage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Nil except on assistant messages.
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type ToolCallData struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Nil for unsupported role or missing ToolCallID.
func (m *Message) ToLLMMessage() *LLMMessage {
	content := ""
	if m.Content != nil {
		content = *m.Content
	}
	switch m.Role {
	case "user":
		msg := UserLLMMessage(content)
		return &msg
	case "assistant":
		var calls []ToolCall
		for _, tc := range m.ToolCalls {
			calls = append(calls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		msg := AssistantLLMMessage(content, calls)
		return &msg
	case "tool":
		if m.ToolCallID == nil {
			return nil
		}
		msg := ToolLLMMessage(content, *m.ToolCallID)
		return &msg
	default:
		return nil
	}
}

type MsgRepo interface {
	Create(ctx context.Context, msg *Message) error
	FindByID(ctx context.Context, id string) (*Message, error)
	ListByConversationID(ctx context.Context, conversationID string) ([]*Message, error)

	MarkModifiedFromTurn(ctx context.Context, conversationID string, turnID int64) error
}

// Redis cache avoids MySQL on prompt assembly.
type MessageCacheRepo interface {
	Push(ctx context.Context, msg *Message) error
	PushAll(ctx context.Context, msgs []*Message) error
	Pop(ctx context.Context, conversationID string) error
	List(ctx context.Context, conversationID string) ([]*Message, error)
	Invalidate(ctx context.Context, conversationID string) error
}

type MsgSeqRepo interface {
	Next(ctx context.Context, conversationID string) (int64, error)
	NextTurn(ctx context.Context, conversationID string) (int64, error)
	NextTurnSeq(ctx context.Context, conversationID string, turnID int64) (int64, error)
}
