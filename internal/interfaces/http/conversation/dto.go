package conversation

import (
	domainconv "learn/internal/domain/conversation"
	"time"
)

type createConversationRequest struct {
	Title string `json:"title" binding:"required,max=64"`
}

type conversationResponse struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	ActiveProject *string `json:"active_project_id,omitempty"`
}

func toConversationResponse(c *domainconv.Conversation) conversationResponse {
	return conversationResponse{
		ID:            c.ID,
		Title:         c.Title,
		ActiveProject: c.ActiveProjectID,
	}
}

type listConversationsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type listConversationsResponse struct {
	Items  []conversationResponse `json:"items"`
	Total  int64                  `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

func toListConversationsResponse(convos []*domainconv.Conversation, total int64, limit, offset int) listConversationsResponse {
	items := make([]conversationResponse, len(convos))
	for i, c := range convos {
		items[i] = toConversationResponse(c)
	}
	return listConversationsResponse{Items: items, Total: total, Limit: limit, Offset: offset}
}

type getConversationPath struct {
	ConversationID string `uri:"conversation_id"`
}
type messageResponse struct {
	ID         string                    `json:"id"`
	Role       string                    `json:"role"`
	Content    string                    `json:"content"`
	TurnID     int64                     `json:"turn_id"`
	SeqID      int64                     `json:"seq_id"`
	Modified   bool                      `json:"is_modified"`
	CreatedAt  time.Time                 `json:"created_at"`
	ToolCalls  []domainconv.ToolCallData `json:"tool_calls,omitempty"`
	ToolCallID *string                   `json:"tool_call_id,omitempty"`
	ToolName   *string                   `json:"tool_name,omitempty"`
}

func toMessageResponse(m *domainconv.Message) messageResponse {
	content := ""
	if m.Content != nil {
		content = *m.Content
	}
	return messageResponse{
		ID:         m.ID,
		Role:       m.Role,
		Content:    content,
		TurnID:     m.TurnID,
		SeqID:      m.SeqID,
		Modified:   m.IsModified,
		CreatedAt:  m.CreatedAt,
		ToolCalls:  m.ToolCalls,
		ToolCallID: m.ToolCallID,
		ToolName:   m.ToolName,
	}
}

func toListMessagesResponse(msgs []*domainconv.Message) []messageResponse {
	out := make([]messageResponse, len(msgs))
	for i, m := range msgs {
		out[i] = toMessageResponse(m)
	}
	return out
}

type listMessagesResponse struct {
	Items           []messageResponse `json:"items"`
	LastTurnAtLoad  int64             `json:"last_turn_at_load"`
	LastSeqIDAtLoad int64             `json:"last_seq_id_at_load"`
	InFlight        bool              `json:"in_flight"`
}
