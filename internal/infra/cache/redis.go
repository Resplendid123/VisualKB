package cache

import (
	"context"
	"fmt"
	"learn/internal/infra/config"

	"github.com/redis/go-redis/v9"
)

func OpenRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, func() error, error) {
	rc := redis.NewClient(&redis.Options{
		Addr:        cfg.Addr,
		Password:    cfg.Password,
		DB:          cfg.DB,
		DialTimeout: cfg.DialTimeout,
	})
	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()
	if err := rc.Ping(pingCtx).Err(); err != nil {
		_ = rc.Close()
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}
	return rc, func() error {
		return rc.Close()
	}, nil
}
