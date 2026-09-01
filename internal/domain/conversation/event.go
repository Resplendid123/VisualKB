package conversation

import (
	"encoding/json"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	TurnID  int64           `json:"turn_id,omitempty"`
	SeqID   int64           `json:"seq_id,omitempty"`
}

// Raw think tags handled by frontend.
type TextPayload struct {
	Delta string `json:"delta"`
}

// Frontend picks rendering by Name.
type ToolCallPayload struct {
	ToolCallID  string         `json:"tool_call_id"`
	Name        string         `json:"name"`
	Args        map[string]any `json:"args"`
	Description string         `json:"description,omitempty"`
}

type ToolResultPayload struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Result     any    `json:"result"`
	Error      string `json:"error,omitempty"`
}

// Emitting question returns placeholder result immediately.
type QuestionPayload struct {
	ToolCallID string      `json:"tool_call_id"`
	Question   string      `json:"question"`
	Options    []AskOption `json:"options"`
}

// Label becomes next user message; Desc UI-only.
type AskOption struct {
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// Turn abort, interrupt, or API error.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type DonePayload struct {
	MessageID string `json:"message_id"`
}
