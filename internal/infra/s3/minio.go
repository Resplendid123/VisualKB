package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"learn/internal/domain/s3"
	"learn/internal/infra/config"
)

type Client struct {
	mc     *minio.Client
	bucket string
}

func New(cfg config.S3Config) (*Client, error) {
	opts := &minio.Options{
		Secure: cfg.UseSSL,
		Region: cfg.Region,
		Creds:  credentials.NewStatic(cfg.AccessKey, cfg.SecretKey, "", credentials.SignatureDefault),
	}
	mc, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("minio new: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func NewWithClient(mc *minio.Client, bucket string) *Client {
	return &Client{mc: mc, bucket: bucket}
}

func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 get %s: %w", key, err)
	}
	return obj, nil
}

func (c *Client) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	_, err := c.mc.PutObject(ctx, c.bucket, key, body, size, opts)
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	return nil
}

func (c *Client) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := c.mc.CopyObject(
		ctx,
		minio.CopyDestOptions{Bucket: c.bucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: c.bucket, Object: srcKey},
	)
	if err != nil {
		return fmt.Errorf("s3 copy %s -> %s: %w", srcKey, dstKey, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("s3 delete %s: %w", key, err)
	}
	return nil
}

func (c *Client) Stat(ctx context.Context, key string) (s3.ObjectInfo, error) {
	st, err := c.mc.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return s3.ObjectInfo{}, fmt.Errorf("s3 stat %s: %w", key, err)
	}
	return s3.ObjectInfo{
		Key:          st.Key,
		Size:         st.Size,
		ContentType:  st.ContentType,
		LastModified: st.LastModified,
	}, nil
}

func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("s3 presign %s: %w", key, err)
	}
	return u.String(), nil
}

var _ s3.ObjectStore = (*Client)(nil)
