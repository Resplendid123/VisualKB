package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"learn/internal/domain/conversation"

	"github.com/redis/go-redis/v9"
)

type eventStreamRepo struct {
	rdb *redis.Client
}

func NewEventStreamRepo(rdb *redis.Client) conversation.EventStreamRepo {
	return &eventStreamRepo{rdb: rdb}
}

func convoKey(id string) string {
	return "events:" + id
}

func (r *eventStreamRepo) Append(ctx context.Context, conversationID string, e conversation.Event) (string, error) {
	key := convoKey(conversationID)

	pipe := r.rdb.Pipeline()
	xAddCmd := pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: conversation.StreamMaxLen,
		Approx: true,
		Values: map[string]any{
			"type":    e.Type,
			"payload": string(e.Payload),
			"turn_id": strconv.FormatInt(e.TurnID, 10),
			"seq_id":  strconv.FormatInt(e.SeqID, 10),
		},
	})

	pipe.Expire(ctx, key, conversation.StreamTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	if xAddCmd.Err() != nil {
		return "", xAddCmd.Err()
	}
	return xAddCmd.Val(), nil
}

func (r *eventStreamRepo) Read(ctx context.Context, conversationID, fromID string, count int64, block time.Duration) ([]conversation.EventRecord, error) {
	key := convoKey(conversationID)

	start := fromID
	if start == "" {
		start = "0"
	}
	res, err := r.rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{key, start},
		Count:   count,
		Block:   block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]conversation.EventRecord, 0)
	for _, s := range res {
		for _, m := range s.Messages {
			typ, _ := m.Values["type"].(string)
			raw, _ := m.Values["payload"].(string)
			out = append(out, conversation.EventRecord{
				ID: m.ID,
				Event: conversation.Event{
					Type:    typ,
					Payload: json.RawMessage(raw),
					TurnID:  parseStreamInt(m.Values["turn_id"]),
					SeqID:   parseStreamInt(m.Values["seq_id"]),
				},
			})
		}
	}
	return out, nil
}

func parseStreamInt(v any) int64 {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case int64:
		return x
	}
	return 0
}

var _ conversation.EventStreamRepo = (*eventStreamRepo)(nil)
