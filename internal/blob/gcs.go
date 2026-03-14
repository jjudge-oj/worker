package blob

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jjudge-oj/worker/config"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSClient wraps the Google Cloud Storage SDK client and bucket name.
type GCSClient struct {
	client    *storage.Client
	bucket    string
	projectID string
}

// NewGCSClient constructs a GCS client from config.
func NewGCSClient(ctx context.Context, cfg *config.GCSConfig) (*GCSClient, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("gcs bucket is required")
	}

	var opts []option.ClientOption
	if strings.TrimSpace(cfg.CredentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &GCSClient{
		client:    client,
		bucket:    cfg.Bucket,
		projectID: cfg.ProjectID,
	}, nil
}

// EnsureBucket ensures the configured bucket exists.
func (g *GCSClient) EnsureBucket(ctx context.Context) error {
	_, err := g.client.Bucket(g.bucket).Attrs(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, storage.ErrBucketNotExist) {
		return err
	}
	if strings.TrimSpace(g.projectID) == "" {
		return errors.New("gcs project id is required to create bucket")
	}
	return g.client.Bucket(g.bucket).Create(ctx, g.projectID, nil)
}

// Put uploads an object to the configured bucket.
func (g *GCSClient) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	writer := g.client.Bucket(g.bucket).Object(key).NewWriter(ctx)
	if strings.TrimSpace(contentType) != "" {
		writer.ContentType = contentType
	}
	if _, err := io.Copy(writer, r); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

// Get opens a reader for an object in the configured bucket.
func (g *GCSClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return g.client.Bucket(g.bucket).Object(key).NewReader(ctx)
}

// Delete removes an object from the configured bucket.
func (g *GCSClient) Delete(ctx context.Context, key string) error {
	return g.client.Bucket(g.bucket).Object(key).Delete(ctx)
}

// Client exposes the underlying GCS SDK client.
func (g *GCSClient) Client() *storage.Client {
	return g.client
}

// Bucket returns the configured bucket name.
func (g *GCSClient) Bucket() string {
	return g.bucket
}

// ProjectID returns the configured project ID.
func (g *GCSClient) ProjectID() string {
	return g.projectID
}
