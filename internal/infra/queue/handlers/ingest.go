package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	documentapp "learn/internal/application/document"
)

type IngestSweepHandler struct {
	svc           *documentapp.DocumentService
	batchLimit    int
	perDocTimeout time.Duration
}

func NewIngestSweepHandler(svc *documentapp.DocumentService, batchLimit int, perDocTimeout time.Duration) *IngestSweepHandler {
	if batchLimit <= 0 {
		batchLimit = 50
	}
	if perDocTimeout <= 0 {
		perDocTimeout = 2 * time.Minute
	}
	return &IngestSweepHandler{svc: svc, batchLimit: batchLimit, perDocTimeout: perDocTimeout}
}

func (h *IngestSweepHandler) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	ids, err := h.svc.ListDirty(ctx, h.batchLimit)
	if err != nil {

		slog.WarnContext(ctx, "list dirty documents failed", "err", err)
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	slog.InfoContext(ctx, "ingest sweep batch", "n", len(ids))
	for _, id := range ids {
		dctx, cancel := context.WithTimeout(ctx, h.perDocTimeout)
		err := h.svc.Ingest(dctx, id)
		cancel()
		if err != nil {

			slog.WarnContext(ctx, "ingest document failed", "document_id", id, "err", err)
		}
	}
	return nil
}
