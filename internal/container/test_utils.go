package container

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/utils"
	"github.com/stretchr/testify/require"
)

const defaultMemoryLimitBytes = 256 * 1024 * 1024

type Testcase struct {
	File  string
	Stdin string

	ExpectedStatus *Status
	ExpectedOutput *string

	TimeLimitUs int64

	Concurrency int
}

func (tc *Testcase) Run(t *testing.T, cfg *config.Config, sp *SlotPool) []*Report {
	t.Helper()

	require.NotNil(t, cfg, "cfg cannot be nil")
	require.NotNil(t, sp, "slot pool cannot be nil")

	prepareRuntimeDirs(t, cfg)

	if tc.Concurrency < 1 {
		tc.Concurrency = 1
	}

	if tc.TimeLimitUs == 0 {
		tc.TimeLimitUs = 1000000
	}

	runDir := filepath.Join(t.TempDir(), uuid.NewString())
	require.NoError(t, os.MkdirAll(runDir, 0755))
	require.NoError(t, utils.FileCopy(tc.File, filepath.Join(runDir, "main.cpp")))

	rootfsImageDir := resolveRootfsImageDir(t, cfg)

	compileReport, err := runCompile(t, cfg, sp, runDir, rootfsImageDir)
	require.NoError(t, err, "compilation failed to start")
	require.Equal(t, STATUS_OK, compileReport.Status, "compile status not ok")

	reports := make([]*Report, tc.Concurrency)
	errCh := make(chan error, tc.Concurrency)

	wg := sync.WaitGroup{}
	for i := 0; i < tc.Concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			report, err := runExecution(t, tc, cfg, sp, runDir, rootfsImageDir)
			if err != nil {
				errCh <- err
				return
			}
			reports[idx] = report

			if tc.ExpectedStatus != nil && report.Status != *tc.ExpectedStatus {
				errCh <- fmt.Errorf("status != expectedStatus: got %s want %s", report.Status, *tc.ExpectedStatus)
			}

			if tc.ExpectedOutput != nil && report.Stdout != *tc.ExpectedOutput {
				errCh <- fmt.Errorf("output != expectedOutput: got %q want %q", report.Stdout, *tc.ExpectedOutput)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	return reports
}

func runCompile(t *testing.T, runtimeCfg *config.Config, sp *SlotPool, workDir, rootfsImageDir string) (*Report, error) {
	t.Helper()

	defaultCompileTimeoutUs := int64(10 * 1000 * 1000)

	return RunInContainer(runtimeCfg, sp, workDir, rootfsImageDir, []string{"g++", "-o", "main", "main.cpp"}, &bytes.Buffer{}, defaultCompileTimeoutUs, defaultMemoryLimitBytes, 64)
}

func runExecution(t *testing.T, tc *Testcase, runtimeCfg *config.Config, sp *SlotPool, workDir, rootfsImageDir string) (*Report, error) {
	t.Helper()

	stdin := &bytes.Buffer{}
	if tc.Stdin != "" {
		stdin.WriteString(tc.Stdin)
	}

	return RunInContainer(runtimeCfg, sp, workDir, rootfsImageDir, []string{"./main"}, stdin, tc.TimeLimitUs, defaultMemoryLimitBytes, 1)
}
