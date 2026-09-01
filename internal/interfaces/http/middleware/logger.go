package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func Logger() gin.HandlerFunc {
	skipPath := map[string]bool{"/metrics": true, "/healthz": true}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if skipPath[path] {
			c.Next()
			return
		}
		start := time.Now()
		method := c.Request.Method
		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		requestID := c.GetString("request_id")

		var errMsg string
		if pub := c.Errors.ByType(gin.ErrorTypePublic).String(); pub != "" {
			errMsg = c.Errors.String()
		} else {
			errMsg = c.Errors.ByType(gin.ErrorTypePrivate).String()
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		attrs := []slog.Attr{
			slog.String("request_id", requestID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
		}
		if errMsg != "" {
			attrs = append(attrs, slog.String("error", errMsg))
		}
		if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
			attrs = append(attrs, slog.String("trace_id", span.SpanContext().TraceID().String()))
		}
		slog.LogAttrs(c.Request.Context(), level, "http request", attrs...)
	}
}
