package lime_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jjudge-oj/worker/internal/lime"
)

const (
	defaultMemoryLimitBytes = 256 * 1024 * 1024
	defaultOutputLimitBytes = 8 * 1024 * 1024
	defaultStackLimitBytes  = 8 * 1024 * 1024
	defaultMaxOpenFiles     = 16
)

type Testcase struct {
	File           string
	Stdin          string
	ExpectedOutput *string
	TimeLimitUs    int64
	Concurrency    int
}

func (tc *Testcase) Run(t *testing.T, rootfsPath string) []*lime.ExecResponse {
	t.Helper()

	limeEnabled := canRunLime(rootfsPath)

	if !limeEnabled {
		t.Skip("set LIME_INTEGRATION=1 and LIME_CGROUP_ROOT, ensure lime is in PATH and rootfs exists")
	}

	if tc.Concurrency < 1 {
		tc.Concurrency = 1
	}
	if tc.TimeLimitUs == 0 {
		tc.TimeLimitUs = 1000000
	}

	runDir := t.TempDir()
	compileInput := filepath.Join(runDir, "compile_input")
	execInput := filepath.Join(runDir, "exec_input")

	if err := os.MkdirAll(compileInput, 0o755); err != nil {
		t.Fatalf("mkdir compile input: %v", err)
	}
	if err := os.MkdirAll(execInput, 0o755); err != nil {
		t.Fatalf("mkdir exec input: %v", err)
	}

	if err := copyFile(tc.File, filepath.Join(compileInput, "main.cpp")); err != nil {
		t.Fatalf("copy source: %v", err)
	}

	idBase := strconv.FormatInt(time.Now().UnixNano(), 10)
	compileReq := makeCompileRequest(idBase, rootfsPath, compileInput, execInput)
	compileResp, err := lime.RunContext(t.Context(), compileReq)
	if err != nil {
		t.Fatalf("compile run: %v", err)
	}
	if compileResp.ExitCode != 0 || compileResp.TermSignal != 0 {
		t.Fatalf("compile failed: exit_code=%d term_signal=%d stderr=%q",
			compileResp.ExitCode, compileResp.TermSignal, compileResp.Stderr)
	}

	if testing.Verbose() {
		b, err := json.MarshalIndent(compileResp, "", "  ")
		if err != nil {
			t.Fatalf("marshal compile response: %v", err)
		}
		t.Logf("Compilation report:\n%s\n", string(b))
	}

	responses := make([]*lime.ExecResponse, tc.Concurrency)
	errCh := make(chan error, tc.Concurrency)
	wg := sync.WaitGroup{}

	for i := 0; i < tc.Concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			execReq := makeExecRequest(idBase, idx, rootfsPath, compileInput, execInput, tc.Stdin, tc.TimeLimitUs)
			resp, err := lime.RunContext(t.Context(), execReq)
			if err != nil {
				errCh <- err
				return
			}
			responses[idx] = &resp

			if tc.ExpectedOutput != nil && resp.Stdout != *tc.ExpectedOutput {
				errCh <- fmt.Errorf("output != expectedOutput: got %q want %q", resp.Stdout, *tc.ExpectedOutput)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("execution error: %v", err)
		}
	}

	for i, resp := range responses {
		if resp == nil {
			t.Fatalf("missing response at index %d", i)
		}

		if testing.Verbose() {
			b, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			t.Logf("exec %d:\n%s\n", i, string(b))
		}
	}

	return responses
}

func canRunLime(rootfsPath string) bool {
	if os.Getenv("LIME_INTEGRATION") == "" {
		return false
	}
	if os.Getenv("LIME_CGROUP_ROOT") == "" {
		return false
	}
	if _, err := exec.LookPath("lime"); err != nil {
		return false
	}
	if _, err := os.Stat(rootfsPath); err != nil {
		return false
	}

	return true
}

func makeCompileRequest(idBase, rootfsPath, compileInput, execInput string) lime.ExecRequest {
	return lime.ExecRequest{
		ID:               "compile-" + idBase,
		Args:             []string{"/usr/local/bin/g++", "-o", "/tmp/output/a.out", "/tmp/input/main.cpp"},
		Envp:             []string{"PATH=/usr/bin:/usr/local/bin:/bin"},
		CPUTimeLimitUs:   10000000,
		WallTimeLimitUs:  20000000,
		MemoryLimitBytes: defaultMemoryLimitBytes,
		MaxProcesses:     64,
		OutputLimitBytes: defaultOutputLimitBytes,
		MaxOpenFiles:     defaultMaxOpenFiles,
		StackLimitBytes:  defaultStackLimitBytes,
		UseCPUs:          "0",
		UseMems:          "0",
		Stdin:            "",
		RootfsPath:       rootfsPath,
		BindMounts: []string{
			compileInput + ":/tmp/input:ro",
			execInput + ":/tmp/output",
		},
		UseOverlayfs: true,
	}
}

func makeExecRequest(idBase string, idx int, rootfsPath, compileInput, execInput, stdin string, timeLimitUs int64) lime.ExecRequest {
	return lime.ExecRequest{
		ID:               fmt.Sprintf("exec-%s-%d", idBase, idx),
		Args:             []string{"/tmp/output/a.out"},
		Envp:             []string{"PATH=/usr/bin:/usr/local/bin:/bin"},
		CPUTimeLimitUs:   uint64(timeLimitUs),
		WallTimeLimitUs:  uint64(timeLimitUs * 2),
		MemoryLimitBytes: defaultMemoryLimitBytes,
		MaxProcesses:     1,
		OutputLimitBytes: defaultOutputLimitBytes,
		MaxOpenFiles:     defaultMaxOpenFiles,
		StackLimitBytes:  defaultStackLimitBytes,
		UseCPUs:          "0",
		UseMems:          "0",
		Stdin:            stdin,
		RootfsPath:       rootfsPath,
		BindMounts: []string{
			compileInput + ":/tmp/input:ro",
			execInput + ":/tmp/output",
		},
		UseOverlayfs: true,
	}
}

var rootfsPath string

func TestMain(m *testing.M) {
	rootfsPath = os.Getenv("LIME_ROOTFS")
	if rootfsPath == "" {
		rootfsPath = "/var/castletown/images/rootfs"
	}
	os.Exit(m.Run())
}

func TestLimeAdd(t *testing.T) {
	expectedOutput := "15\n"
	tc := Testcase{
		File:           filepathJoin("add.cpp"),
		Stdin:          "6 9\n",
		ExpectedOutput: &expectedOutput,
		TimeLimitUs:    1000000,
	}

	tc.Run(t, rootfsPath)
}

func TestLimeTimeLimitExceededA(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("tl1.cpp"),
		TimeLimitUs: 1000000,
	}

	tc.Run(t, rootfsPath)
}

func TestLimeTimeLimitExceededB(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("printloop.cpp"),
		TimeLimitUs: 1000000,
	}

	tc.Run(t, rootfsPath)
}

func TestLimeMemoryLimitExceeded(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("mem1.cpp"),
		TimeLimitUs: 10000000,
	}

	tc.Run(t, rootfsPath)
}

func TestLimeFork(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("fork.cpp"),
		TimeLimitUs: 1000000,
	}

	tc.Run(t, rootfsPath)
}

func TestLimeRusageConsistency(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("random.cpp"),
		TimeLimitUs: 1000000,
	}

	var minCPU, maxCPU uint64
	for i := 0; i < 10; i++ {
		reports := tc.Run(t, rootfsPath)
		report := reports[0]

		if i == 0 {
			minCPU = report.CPUTimeUs
			maxCPU = report.CPUTimeUs
			continue
		}
		if report.CPUTimeUs < minCPU {
			minCPU = report.CPUTimeUs
		}
		if report.CPUTimeUs > maxCPU {
			maxCPU = report.CPUTimeUs
		}
	}

	if maxCPU-minCPU >= 10000 {
		t.Fatalf("cpu usage inconsistent: min=%d max=%d", minCPU, maxCPU)
	}
}

func TestLimeConcurrency(t *testing.T) {
	tc := Testcase{
		File:        filepathJoin("sleep.cpp"),
		TimeLimitUs: 3000000,
		Concurrency: 5,
	}

	tc.Run(t, rootfsPath)
}

func filepathJoin(name string) string {
	return filepath.Join("test_files", name)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return fmt.Errorf("copy failed: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("close destination: %w", err)
	}

	return nil
}
