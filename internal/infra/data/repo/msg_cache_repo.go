package repo

import (
	"context"
	"encoding/json"
	"learn/internal/domain/conversation"

	"github.com/redis/go-redis/v9"
)

type msgCacheRepo struct {
	rdb *redis.Client
}

func NewMsgCacheRepo(rdb *redis.Client) conversation.MessageCacheRepo {
	return &msgCacheRepo{rdb: rdb}
}

func msgCacheKey(conversationID string) string {
	return "msgs:" + conversationID
}

func (r *msgCacheRepo) Push(ctx context.Context, msg *conversation.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	key := msgCacheKey(msg.ConversationID)

	pipe := r.rdb.Pipeline()
	pipe.RPush(ctx, key, payload)
	pipe.Expire(ctx, key, conversation.MessageCacheTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *msgCacheRepo) PushAll(ctx context.Context, msgs []*conversation.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	key := msgCacheKey(msgs[0].ConversationID)

	values := make([]any, len(msgs))
	for i, m := range msgs {
		payload, err := json.Marshal(m)
		if err != nil {
			return err
		}
		values[i] = payload
	}

	pipe := r.rdb.Pipeline()
	pipe.RPush(ctx, key, values...)
	pipe.Expire(ctx, key, conversation.MessageCacheTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *msgCacheRepo) Pop(ctx context.Context, conversationID string) error {

	key := msgCacheKey(conversationID)
	return r.rdb.LTrim(ctx, key, 0, -2).Err()
}

func (r *msgCacheRepo) List(ctx context.Context, conversationID string) ([]*conversation.Message, error) {
	raw, err := r.rdb.LRange(ctx, msgCacheKey(conversationID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]*conversation.Message, 0, len(raw))
	for _, s := range raw {
		var m conversation.Message
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, nil
}

func (r *msgCacheRepo) Invalidate(ctx context.Context, conversationID string) error {
	return r.rdb.Del(ctx, msgCacheKey(conversationID)).Err()
}

var _ conversation.MessageCacheRepo = (*msgCacheRepo)(nil)
