package response

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SSEWriter struct {
	writer  io.Writer
	flusher http.Flusher
	mu      sync.Mutex
	done    chan struct{}
	closed  bool
}

func NewSSEWriter(w gin.ResponseWriter) *SSEWriter {
	flusher, _ := w.(http.Flusher)
	return &SSEWriter{
		writer:  w,
		flusher: flusher,
		done:    make(chan struct{}),
	}
}

func WriteSSEHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
}

// Ping writes an SSE keep-alive comment.
func (s *SSEWriter) Ping() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = fmt.Fprintf(s.writer, ": ping\n\n")
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// WriteEvent writes a single SSE frame.
func (s *SSEWriter) WriteEvent(eventType string, payload json.RawMessage, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := fmt.Fprintf(s.writer, "event: %s\n", eventType); err != nil {
		return err
	}
	if eventID != "" {
		if _, err := fmt.Fprintf(s.writer, "id: %s\n", eventID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(s.writer, "data: %s\n\n", string(payload)); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

func (s *SSEWriter) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	// Clear flusher; nil flush panics post-close.
	s.flusher = nil
	close(s.done)
}

func (s *SSEWriter) StartHeartbeat(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Ping()
			case <-s.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}
