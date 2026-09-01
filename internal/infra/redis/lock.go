package redisinfra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Key prefix for all session locks.
const SessionLockPrefix = "lock:session:"

var (
	// SET NX failed, lock held elsewhere.
	ErrLockNotAcquired = errors.New("session lock not acquired")
	// Owner mismatch (held or expired).
	ErrLockNotHeld = errors.New("session lock not held by caller")
)

// Atomic release: owner-check + DEL.
const releaseLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end`

// Atomic refresh: owner-check + EXPIRE.
const refreshLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
	return 0
end`

// Per-conv lock; TTL caps agent ctx.
type SessionLock struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionLock(rdb *redis.Client, ttl time.Duration) *SessionLock {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &SessionLock{rdb: rdb, ttl: ttl}
}

// Try acquire lock as owner.
func (s *SessionLock) Acquire(ctx context.Context, convID, owner string) (*Lock, error) {
	if convID == "" {
		return nil, fmt.Errorf("session lock acquire: empty conv id")
	}
	if owner == "" {
		return nil, fmt.Errorf("session lock acquire: empty owner")
	}
	key := SessionLockPrefix + convID
	ok, err := s.rdb.SetNX(ctx, key, owner, s.ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("session lock acquire: %w", err)
	}
	if !ok {
		return nil, ErrLockNotAcquired
	}
	return &Lock{owner: owner, key: key}, nil
}

// Current owner; empty if unlocked.
func (s *SessionLock) CurrentOwner(ctx context.Context, convID string) (string, error) {
	v, err := s.rdb.Get(ctx, SessionLockPrefix+convID).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

// Lock present (any owner).
func (s *SessionLock) IsHeld(ctx context.Context, convID string) (bool, error) {
	n, err := s.rdb.Exists(ctx, SessionLockPrefix+convID).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Renew TTL when owner matches.
func (s *SessionLock) Heartbeat(ctx context.Context, convID, owner string) error {
	if convID == "" || owner == "" {
		return fmt.Errorf("session lock heartbeat: empty conv id / owner")
	}
	res, err := s.rdb.Eval(ctx, refreshLockScript,
		[]string{SessionLockPrefix + convID}, owner, int(s.ttl.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("session lock heartbeat: %w", err)
	}
	n, ok := res.(int64)
	if !ok || n == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// Release lock; idempotent owner-check.
func (s *SessionLock) Release(ctx context.Context, convID, owner string) error {
	if convID == "" || owner == "" {
		return fmt.Errorf("session lock release: empty conv id / owner")
	}
	res, err := s.rdb.Eval(ctx, releaseLockScript,
		[]string{SessionLockPrefix + convID}, owner).Result()
	if err != nil {
		return fmt.Errorf("session lock release: %w", err)
	}
	n, ok := res.(int64)
	if !ok || n == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// Lock handle for defer release.
type Lock struct {
	owner string
	key   string
}

func (l *Lock) Owner() string { return l.owner }
