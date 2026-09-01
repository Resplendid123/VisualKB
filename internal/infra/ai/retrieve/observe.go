package retrieve

import (
	"context"
	"log/slog"
	"time"
)

func logPhase(ctx context.Context, stage string, start time.Time, fields ...any) {
	elapsed := time.Since(start)
	attrs := make([]any, 0, len(fields)+4)
	attrs = append(attrs, "phase", "rag."+stage, "elapsed_ms", elapsed.Milliseconds())
	attrs = append(attrs, fields...)
	slog.InfoContext(ctx, "rag phase", attrs...)
}

func errStatus(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}
