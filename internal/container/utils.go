package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/joshjms/castletown/internal/config"
	"github.com/stretchr/testify/require"
)

func RunInContainer(runtimeCfg *config.Config, sp *SlotPool, workDir, rootfsImageDir string, args []string, stdin io.Reader, timeLimitUs int64, memoryLimitBytes int64, maxProcs int64) (*Report, error) {
	allocation, err := sp.Allocate(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to allocate slot: %w", err)
	}

	cfg := UseDefaultConfig()
	cfg.Args = args
	cfg.Stdin = stdin
	cfg.BindMount = workDir
	cfg.RootfsImageDir = rootfsImageDir
	cfg.TimeLimitUs = timeLimitUs
	cfg.MemoryLimitBytes = memoryLimitBytes
	cfg.PidLimit = maxProcs

	overlay, err := newOverlay(runtimeCfg, rootfsImageDir)
	if err != nil {
		allocation.Release()
		return nil, err
	}
	cfg.Overlay = overlay
	cfg.UseAllocation(allocation)

	return runContainer(context.Background(), runtimeCfg, cfg)
}

func runContainer(ctx context.Context, runtimeCfg *config.Config, cfg *Config) (*Report, error) {
	container := NewContainer("", runtimeCfg, WithContainerConfig(cfg))

	if err := container.Init(ctx); err != nil {
		destroyErr := container.Destroy()
		if destroyErr != nil {
			return nil, fmt.Errorf("init error (%w) and destroy error (%v)", err, destroyErr)
		}

		return nil, fmt.Errorf("failed to init container: %w", err)
	}

	report, runErr := container.Run(ctx)
	destroyErr := container.Destroy()

	if runErr != nil {
		return report, fmt.Errorf("failed to run container: %w", runErr)
	}

	if destroyErr != nil {
		return report, fmt.Errorf("failed to destroy container: %w", destroyErr)
	}

	return report, nil
}

func newOverlay(runtimeCfg *config.Config, rootfsImageDir string) (*Overlay, error) {
	base := filepath.Join(runtimeCfg.OverlayFSDir, uuid.NewString())
	overlay, err := NewOverlay(base, rootfsImageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create overlay: %w", err)
	}

	return overlay, nil
}

func resolveRootfsImageDir(t *testing.T, cfg *config.Config) string {
	t.Helper()

	defaultImagePath := filepath.Join(cfg.ImagesDir, "gcc-15-bookworm")
	if _, err := os.Stat(defaultImagePath); err == nil {
		return defaultImagePath
	}

	entries, err := os.ReadDir(cfg.ImagesDir)
	require.NoError(t, err, "failed to read images dir %s", cfg.ImagesDir)

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(cfg.ImagesDir, entry.Name())
		}
	}

	t.Fatalf("no rootfs image directory found in %s; run scripts/rootfs.sh to populate one", cfg.ImagesDir)
	return ""
}

func prepareRuntimeDirs(t *testing.T, cfg *config.Config) {
	t.Helper()

	require.NoError(t, os.MkdirAll(cfg.LibcontainerDir, 0o755))
	require.NoError(t, os.MkdirAll(cfg.OverlayFSDir, 0o755))
}
