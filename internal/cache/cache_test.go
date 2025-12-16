package cache

import (
	"context"
	"testing"

	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/repository"
	"github.com/joshjms/castletown/internal/store"
	"github.com/stretchr/testify/require"
)

// TestProblemCache tests the basic functionality of the ProblemCache. It is not used in CI and should only be run locally.
func TestProblemCache(t *testing.T) {
	cfg := config.Load()
	pr, err := repository.NewProblemsRepository(cfg.Database)
	require.NoError(t, err, "cannot create postgres repository: %v", err)

	tcs, err := store.NewTestcaseStore(cfg.Minio)
	require.NoError(t, err, "cannot create minio store: %v", err)

	var problemID = 2

	pc := NewProblemCache(pr, tcs, 5, cfg.DiskCacheDir)
	require.NotNil(t, pc, "problem cache should not be nil")

	_, release, err := pc.GetProblemWithLease(context.Background(), problemID)
	require.NoError(t, err, "error getting problem with lease: %v", err)
	release()
}
