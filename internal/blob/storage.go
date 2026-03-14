package blob

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jjudge-oj/worker/config"
)

// ObjectStorage defines common object operations across backends.
type ObjectStorage interface {
	EnsureBucket(ctx context.Context) error
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Bucket() string
}

// Storage wraps an ObjectStorage backend with a stable API.
type Storage struct {
	backend ObjectStorage
}

// NewStorage constructs a Storage wrapper for the provided backend.
func NewStorage(backend ObjectStorage) *Storage {
	return &Storage{backend: backend}
}

// NewStorageFromConfig selects an object storage backend based on config.
func NewStorageFromConfig(ctx context.Context, cfg *config.Config) (*Storage, error) {
	if strings.TrimSpace(cfg.GCS.Bucket) != "" {
		client, err := NewGCSClient(ctx, cfg.GCS)
		if err != nil {
			return nil, err
		}
		return NewStorage(client), nil
	}

	if strings.TrimSpace(cfg.Minio.AccessKey) != "" || strings.TrimSpace(cfg.Minio.SecretKey) != "" {
		client, err := NewMinioClient(cfg.Minio)
		if err != nil {
			return nil, err
		}
		return NewStorage(client), nil
	}

	return nil, errors.New("object storage configuration is required")
}

// EnsureBucket ensures the configured bucket exists.
func (s *Storage) EnsureBucket(ctx context.Context) error {
	return s.backend.EnsureBucket(ctx)
}

// Put uploads an object to the configured bucket.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	return s.backend.Put(ctx, key, r, size, contentType)
}

// Get opens a reader for an object in the configured bucket.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.backend.Get(ctx, key)
}

// Delete removes an object from the configured bucket.
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.backend.Delete(ctx, key)
}

// Bucket returns the configured bucket name.
func (s *Storage) Bucket() string {
	return s.backend.Bucket()
}
