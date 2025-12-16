package store

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/joshjms/castletown/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type TestcaseStore struct {
	client *minio.Client
}

func (tcs *TestcaseStore) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := tcs.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, err
	}

	return obj, nil
}

func (tcs *TestcaseStore) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
	return tcs.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
}

// NewTestcaseStore builds a MinIO client from config and returns an ObjectStore implementation.
func NewTestcaseStore(cfg config.MinioConfig) (*TestcaseStore, error) {
	client, err := newClient(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.UseSSL)
	if err != nil {
		return nil, err
	}
	return &TestcaseStore{client: client}, nil
}

func newClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
}
