// Package s3 is the object store port.
package s3

import (
	"context"
	"io"
	"time"
)

type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

// Key format is caller-defined, e.g. "documents/123/v1.md".
type ObjectStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error

	Copy(ctx context.Context, srcKey, dstKey string) error

	Delete(ctx context.Context, key string) error

	Stat(ctx context.Context, key string) (ObjectInfo, error)

	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}
