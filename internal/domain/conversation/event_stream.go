package conversation

import (
	"context"
	"time"
)

const (
	EventTypeText     = "text"
	EventTypeToolCall = "tool_call"
	EventTypeToolRes  = "tool_result"
	EventTypeQuestion = "question"
	EventTypeError    = "error"
	EventTypeDone     = "done"
)

const (
	StreamMaxLen       int64 = 1000             // max entries per key; FIFO evict
	StreamTTL                = 15 * time.Minute // per-key TTL
	StreamBatchSize    int64 = 32               // max entries per XRead
	StreamBlockTimeout       = 1 * time.Second
)

// ID is the Redis stream entry id.
type EventRecord struct {
	ID    string
	Event Event
}

type EventStreamRepo interface {
	Append(ctx context.Context, conversationID string, e Event) (string, error)
	Read(ctx context.Context, conversationID, fromID string, count int64, block time.Duration) ([]EventRecord, error)
}
