package conversation

import "context"

const DefaultMaxRound = 60

type LLMClient interface {
	Stream(
		ctx context.Context,
		msgs []LLMMessage,
		tools []Tool,
		onText func(string),
	) (string, []ToolCall, *TokenUsage, error)
}

// Events go to Redis Stream and SSE.
type EventSink interface {
	Emit(ctx context.Context, e Event) error
}

type MessageSink interface {
	AllocTurnSeq(ctx context.Context, conversationID string, turnID int64) (int64, error)
	Persist(ctx context.Context, m *Message, turnSeq int64) (int64, error)
	AbortSysMsg(ctx context.Context, turnID int64) error
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Vendor-neutral message format, decoupled from SDKs.
type LLMMessage struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID *string
}

func SystemLLMMessage(content string) LLMMessage {
	return LLMMessage{Role: "system", Content: content}
}

func UserLLMMessage(content string) LLMMessage {
	return LLMMessage{Role: "user", Content: content}
}

func AssistantLLMMessage(content string, calls []ToolCall) LLMMessage {
	return LLMMessage{Role: "assistant", Content: content, ToolCalls: calls}
}

func ToolLLMMessage(content, toolCallID string) LLMMessage {
	id := toolCallID
	return LLMMessage{Role: "tool", Content: content, ToolCallID: &id}
}
