package repo

import (
	"context"
	"encoding/json"

	"learn/internal/domain/conversation"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type msgRepo struct {
	db *gorm.DB
}

func NewMsgRepo(db *gorm.DB) conversation.MsgRepo {
	return &msgRepo{db: db}
}

func (r *msgRepo) Create(ctx context.Context, msg *conversation.Message) error {
	m, err := msgToModel(msg)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}

	msg.ID = m.ID
	return nil
}

func (r *msgRepo) FindByID(ctx context.Context, id string) (*conversation.Message, error) {

	var m model.Message
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return msgToDomain(&m)
}

func (r *msgRepo) ListByConversationID(ctx context.Context, conversationID string) ([]*conversation.Message, error) {

	var ms []model.Message
	err := r.db.WithContext(ctx).
		Table("messages").
		Select("messages.*").
		Joins("JOIN conversations ON conversations.id = messages.conversation_id").
		Where("messages.conversation_id = ?::uuid", conversationID).
		Where("messages.is_modified = ?", false).
		Where("conversations.last_compression_at IS NULL OR messages.created_at > conversations.last_compression_at").
		Order("messages.turn_id ASC, messages.seq_id ASC").
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*conversation.Message, 0, len(ms))
	for i := range ms {
		d, err := msgToDomain(&ms[i])
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *msgRepo) MarkModifiedFromTurn(ctx context.Context, conversationID string, turnID int64) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE messages SET is_modified = true
		WHERE conversation_id = ?::uuid AND turn_id = ?`, conversationID, turnID).Error
}

func msgToDomain(m *model.Message) (*conversation.Message, error) {
	var calls []conversation.ToolCallData
	if len(m.ToolCalls) > 0 {
		if err := json.Unmarshal(m.ToolCalls, &calls); err != nil {
			return nil, err
		}
	}
	var usage *conversation.TokenUsage
	if m.PromptTokens != nil || m.CompletionTokens != nil {
		usage = &conversation.TokenUsage{}
		if m.PromptTokens != nil {
			usage.PromptTokens = *m.PromptTokens
		}
		if m.CompletionTokens != nil {
			usage.CompletionTokens = *m.CompletionTokens
		}
	}
	return &conversation.Message{
		ID:               m.ID,
		ConversationID:   m.ConversationID,
		Role:             m.Role,
		Content:          m.Content,
		Seq:              m.Seq,
		TurnID:           m.TurnID,
		SeqID:            m.SeqID,
		IsModified:       m.IsModified,
		IsCompressed:     m.IsCompressed,
		IsToolCompressed: m.IsToolCompressed,
		ToolCalls:        calls,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		TokenUsage:       usage,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}, nil
}

func msgToModel(msg *conversation.Message) (*model.Message, error) {
	var raw []byte
	if len(msg.ToolCalls) > 0 {
		b, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	var pt, ot *int64
	if msg.TokenUsage != nil {
		if msg.TokenUsage.PromptTokens > 0 {
			v := msg.TokenUsage.PromptTokens
			pt = &v
		}
		if msg.TokenUsage.CompletionTokens > 0 {
			v := msg.TokenUsage.CompletionTokens
			ot = &v
		}
	}
	return &model.Message{
		ID:               msg.ID,
		ConversationID:   msg.ConversationID,
		Role:             msg.Role,
		Content:          msg.Content,
		Seq:              msg.Seq,
		TurnID:           msg.TurnID,
		SeqID:            msg.SeqID,
		IsModified:       msg.IsModified,
		IsCompressed:     msg.IsCompressed,
		IsToolCompressed: msg.IsToolCompressed,
		ToolCalls:        raw,
		ToolCallID:       msg.ToolCallID,
		ToolName:         msg.ToolName,
		PromptTokens:     pt,
		CompletionTokens: ot,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
	}, nil
}

var _ conversation.MsgRepo = (*msgRepo)(nil)
