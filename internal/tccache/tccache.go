package tccache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jjudge-oj/worker/internal/blob"
)

type TestcaseCache struct {
	cache *LRUCache[string, bool]

	tcCacheDir string
	blob       *blob.Storage
}

func NewTestcaseCache(capacity int, cacheDir string) (*TestcaseCache, error) {
	tcCacheDir := filepath.Join(cacheDir, "testcases")
	if err := os.MkdirAll(tcCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create testcase cache directory: %w", err)
	}

	onDelete := func(key string) {
		tcPath := filepath.Join(tcCacheDir, filepath.FromSlash(key))
		if err := os.Remove(tcPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("failed to remove testcase file %s: %v\n", tcPath, err)
		}
	}

	return &TestcaseCache{
		cache:      New[string, bool](capacity, onDelete),
		tcCacheDir: tcCacheDir,
	}, nil
}

func (tcc *TestcaseCache) Get(key string) (string, bool) {
	if _, ok := tcc.cache.Get(key); !ok {
		return "", false
	}
	return filepath.Join(tcc.tcCacheDir, filepath.FromSlash(key)), true
}

// SetBlobStorage sets the blob storage backend used for fetching testcases.
func (tcc *TestcaseCache) SetBlobStorage(b *blob.Storage) {
	tcc.blob = b
}

// GetOrFetch returns the local file path for a testcase, fetching from object storage if not cached.
func (tcc *TestcaseCache) GetOrFetch(ctx context.Context, key string) (string, error) {
	tcPath := filepath.Join(tcc.tcCacheDir, filepath.FromSlash(key))
	if _, err := os.Stat(tcPath); err == nil {
		return tcPath, nil
	}

	r, err := tcc.blob.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to get testcase from object storage: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(filepath.Dir(tcPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create testcase cache directory: %w", err)
	}

	f, err := os.Create(tcPath)
	if err != nil {
		return "", fmt.Errorf("failed to create testcase file: %w", err)
	}
	defer f.Close()

	if _, err := f.ReadFrom(r); err != nil {
		return "", fmt.Errorf("failed to write testcase to file: %w", err)
	}

	tcc.cache.Put(key, true)
	return tcPath, nil
}
