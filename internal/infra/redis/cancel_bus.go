package redisinfra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Payload for cross-instance turn cancel.
type CancelMessage struct {
	ConvID string `json:"conv_id"`
	Reason string `json:"reason,omitempty"`
}

// Derive cancel channel from owner prefix.
func channelForOwner(owner string) (string, error) {
	if owner == "" {
		return "", errors.New("cancel bus: empty owner")
	}
	i := strings.IndexByte(owner, ':')
	if i <= 0 {
		return "", fmt.Errorf("cancel bus: malformed owner %q (no instance prefix)", owner)
	}
	return "cancel:" + owner[:i], nil
}

// Cross-instance cancel pub/sub forwarder.
type CancelBus struct {
	rdb        *redis.Client
	instanceID string
}

// Build channel name from instanceID.
func NewCancelBus(rdb *redis.Client, instanceID string) *CancelBus {
	return &CancelBus{rdb: rdb, instanceID: instanceID}
}

// Current channel name (debug/test).
func (b *CancelBus) Channel() string { return "cancel:" + b.instanceID }

// Send cancel to owner's instance channel.
func (b *CancelBus) Publish(ctx context.Context, targetOwner string, msg CancelMessage) error {
	ch, err := channelForOwner(targetOwner)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("cancel bus: marshal: %w", err)
	}
	return b.rdb.Publish(ctx, ch, payload).Err()
}

// Stream cancel signals for this instance.
func (b *CancelBus) Subscribe(ctx context.Context) (<-chan CancelMessage, func() error) {
	out := make(chan CancelMessage, 16)
	sub := b.rdb.Subscribe(ctx, b.Channel())

	go func() {
		defer close(out)
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var cm CancelMessage
				if err := json.Unmarshal([]byte(msg.Payload), &cm); err != nil {
					slog.WarnContext(ctx, "cancel bus: unmarshal failed",
						"err", err, "channel", b.Channel())
					continue
				}
				select {
				case out <- cm:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	cleanup := func() error { return sub.Close() }
	return out, cleanup
}
