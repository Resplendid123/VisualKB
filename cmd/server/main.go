package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"learn/internal/infra/config"
	"learn/internal/infra/observe"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("load configs failed: %v", err)
	}

	cleanupObserve, err := observe.Load(ctx, cfg.Observe, cfg.Logging)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load observe or logging failed:", err)
		slog.Error("load observe or logging failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := cleanupObserve(); err != nil {
			slog.Error("cleanup observe failed", "err", err)
		}
	}()

	app, err := Bootstrap(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap failed:", err)
		slog.Error("bootstrap failed", "err", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	go func() {
		if err := app.HTTP.Start(); err != nil {
			slog.Error("server start failed", "err", err)
			cancel()
		}
	}()
	<-ctx.Done()
	slog.Info("received shutdown signal, gracefully shutting down...")
}
