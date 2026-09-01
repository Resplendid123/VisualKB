package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

const (
	defaultConcurrency     = 10
	defaultShutdownTimeout = 30 * time.Second
)

type Config struct {
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	Concurrency     int
	ShutdownTimeout time.Duration
}

type Queue struct {
	server    *asynq.Server
	scheduler *asynq.Scheduler
	mux       *asynq.ServeMux
	cfg       Config
}

func New(cfg Config) (*Queue, error) {
	if cfg.RedisAddr == "" {
		return nil, errors.New("queue: redis addr required")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConcurrency
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     cfg.Concurrency,
		ShutdownTimeout: cfg.ShutdownTimeout,
		Queues: map[string]int{
			"default": 1,
		},

		RetryDelayFunc: defaultRetryDelay,
	})

	sched := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		EnqueueErrorHandler: func(task *asynq.Task, _ []asynq.Option, err error) {
			slog.Warn("scheduler enqueue failed", "type", task.Type(), "err", err)
		},
	})

	return &Queue{
		server:    srv,
		scheduler: sched,
		mux:       asynq.NewServeMux(),
		cfg:       cfg,
	}, nil
}

func defaultRetryDelay(n int, _ error, _ *asynq.Task) time.Duration {
	const base, max = 2 * time.Second, 60 * time.Second
	if n < 1 {
		return base
	}
	if n > 5 {
		return max
	}
	d := min(base*(1<<(n-1)), max)
	return d
}

func (q *Queue) Mux() *asynq.ServeMux { return q.mux }

func (q *Queue) Scheduler() *asynq.Scheduler { return q.scheduler }

func (q *Queue) Start(ctx context.Context) error {
	if err := q.server.Start(q.mux); err != nil {
		return fmt.Errorf("queue server start: %w", err)
	}
	if err := q.scheduler.Start(); err != nil {
		q.server.Shutdown()
		return fmt.Errorf("queue scheduler start: %w", err)
	}
	slog.InfoContext(ctx, "queue started", "concurrency", q.cfg.Concurrency, "shutdown_timeout", q.cfg.ShutdownTimeout)
	return nil
}

func (q *Queue) Shutdown() {
	q.scheduler.Shutdown()
	q.server.Shutdown()
}
