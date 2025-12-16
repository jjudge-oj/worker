package container_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/joshjms/castletown/internal/config"
	"github.com/joshjms/castletown/internal/container"
	"github.com/stretchr/testify/require"
)

var sp *container.SlotPool
var cfg *config.Config

func TestMain(m *testing.M) {
	container.Init()
	cfg = config.Load()
	cfg.Judge.MaxConcurrency = 2

	if os.Geteuid() == 0 {
		files, err := os.ReadDir("test_files")
		require.NoError(nil, err, "failed to read test files directory: %v", err)

		for _, f := range files {
			fullPath := filepath.Join("test_files", f.Name())
			if err := os.Chown(fullPath, 0, 0); err != nil {
				panic(err)
			}
		}
	}

	sp = container.NewSlotPool(container.WithMaxConcurrency(cfg.Judge.MaxConcurrency))

	exitCode := m.Run()

	if os.Geteuid() == 0 {
		files, err := os.ReadDir("test_files")
		require.NoError(nil, err, "failed to read test files directory: %v", err)

		for _, f := range files {
			fullPath := filepath.Join("test_files", f.Name())
			if err := os.Chown(fullPath, 1000, 1000); err != nil {
				panic(err)
			}
		}
	}

	os.Exit(exitCode)
}

func TestContainerAdd(t *testing.T) {
	expectedStatus := container.STATUS_OK
	expectedOutput := "15\n"

	tc := container.Testcase{
		File:           "test_files/add.cpp",
		Stdin:          "6 9\n",
		ExpectedStatus: &expectedStatus,
		ExpectedOutput: &expectedOutput,
		TimeLimitUs:    1000000,
	}

	tc.Run(t, cfg, sp)
}

func TestContainerTimeLimitExceededA(t *testing.T) {
	expectedStatus := container.STATUS_TIME_LIMIT_EXCEEDED

	tc := container.Testcase{
		File:           "test_files/tl1.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    1000000,
	}

	tc.Run(t, cfg, sp)
}

func TestContainerTimeLimitExceededB(t *testing.T) {
	expectedStatus := container.STATUS_TIME_LIMIT_EXCEEDED

	tc := container.Testcase{
		File:           "test_files/printloop.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    1000000,
	}

	tc.Run(t, cfg, sp)
}

func TestContainerMemoryLimitExceeded(t *testing.T) {
	expectedStatus := container.STATUS_MEMORY_LIMIT_EXCEEDED

	tc := container.Testcase{
		File:           "test_files/mem1.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    10000000,
	}

	tc.Run(t, cfg, sp)
}

func TestContainerFork(t *testing.T) {
	expectedStatus := container.STATUS_OK

	tc := container.Testcase{
		File:           "test_files/fork.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    1000000,
	}

	tc.Run(t, cfg, sp)
}

func TestContainerRusageConsistency(t *testing.T) {
	expectedStatus := container.STATUS_OK

	tc := container.Testcase{
		File:           "test_files/random.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    1000000,
	}

	var minCpuUsage, maxCpuUsage uint64

	for i := 0; i < 10; i++ {
		reports := tc.Run(t, cfg, sp)
		report := reports[0]

		if i == 0 {
			minCpuUsage = report.CPUTime
			maxCpuUsage = report.CPUTime

			continue
		}

		minCpuUsage = min(minCpuUsage, report.CPUTime)
		maxCpuUsage = max(maxCpuUsage, report.CPUTime)
	}

	require.Less(t, maxCpuUsage-minCpuUsage, uint64(10000), "cpu usage inconsistent")
}

func TestContainerConcurrency(t *testing.T) {
	expectedStatus := container.STATUS_OK

	tc := container.Testcase{
		File:           "test_files/sleep.cpp",
		ExpectedStatus: &expectedStatus,
		TimeLimitUs:    3000000,
		Concurrency:    5,
	}

	reports := tc.Run(t, cfg, sp)

	startTimes := make([]int64, len(reports))
	finishTimes := make([]int64, len(reports))

	for i, report := range reports {
		startTimes[i] = report.StartAt.UnixMilli()
		finishTimes[i] = report.FinishAt.UnixMilli()
	}

	sort.Slice(startTimes, func(i, j int) bool {
		return startTimes[i] < startTimes[j]
	})
	sort.Slice(finishTimes, func(i, j int) bool {
		return finishTimes[i] < finishTimes[j]
	})

	for i := 2; i < len(startTimes); i++ {
		require.Less(t, finishTimes[i-2], startTimes[i], "semaphore didn't work correctly")
	}

	tc.Run(t, cfg, sp)
}
