package lime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jjudge-oj/worker/config"
)

type ExecRequest struct {
	ID               string   `json:"id"`
	Args             []string `json:"args"`
	Envp             []string `json:"envp"`
	CPUTimeLimitUs   uint64   `json:"cpu_time_limit_us"`
	WallTimeLimitUs  uint64   `json:"wall_time_limit_us"`
	MemoryLimitBytes uint64   `json:"memory_limit_bytes"`
	MaxProcesses     uint32   `json:"max_processes"`
	OutputLimitBytes uint64   `json:"output_limit_bytes"`
	MaxOpenFiles     uint32   `json:"max_open_files"`
	StackLimitBytes  uint64   `json:"stack_limit_bytes"`
	UseCPUs          string   `json:"use_cpus"`
	UseMems          string   `json:"use_mems"`
	Stdin            string   `json:"stdin"`
	RootfsPath       string   `json:"rootfs_path"`
	BindMounts       []string `json:"bind_mounts"`
	UseOverlayfs     bool     `json:"use_overlayfs"`
}

type ExecResponse struct {
	ID          string `json:"id"`
	ExitCode    int    `json:"exit_code"`
	TermSignal  int    `json:"term_signal"`
	CPUTimeUs   uint64 `json:"cpu_time_us"`
	WallTimeUs  uint64 `json:"wall_time_us"`
	MemoryBytes uint64 `json:"memory_bytes"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
}

const (
	defaultOutputLimitBytes = 8 * 1024 * 1024
	defaultStackLimitBytes  = 8 * 1024 * 1024
	defaultMaxOpenFiles     = 16
)

func Run(ctx context.Context, runtimeCfg *config.Config, sp *SlotPool, workDir, rootfsPath string, args []string, stdin string, timeLimitUs uint64, memoryLimitBytes uint64, maxProcs uint32) (*Report, error) {
	if sp == nil {
		return nil, fmt.Errorf("slot pool is nil")
	}

	allocation, err := sp.Allocate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate slot: %w", err)
	}
	defer allocation.Release()

	rootfsPath = strings.TrimSpace(rootfsPath)
	if rootfsPath == "" {
		rootfsPath = runtimeCfg.Judge.RootfsDir
	}

	absRootfs, err := filepath.Abs(rootfsPath)
	if err != nil {
		return nil, fmt.Errorf("resolve rootfs path: %w", err)
	}
	if _, err := os.Stat(absRootfs); err != nil {
		return nil, fmt.Errorf("rootfs path not found: %w", err)
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve work dir: %w", err)
	}

	wallTimeUs := timeLimitUs * 2

	req := ExecRequest{
		ID:               uuid.NewString(),
		Args:             args,
		Envp:             []string{"PATH=/usr/bin:/usr/local/bin:/bin"},
		CPUTimeLimitUs:   timeLimitUs,
		WallTimeLimitUs:  wallTimeUs,
		MemoryLimitBytes: memoryLimitBytes,
		MaxProcesses:     maxProcs,
		OutputLimitBytes: defaultOutputLimitBytes,
		MaxOpenFiles:     defaultMaxOpenFiles,
		StackLimitBytes:  defaultStackLimitBytes,
		UseCPUs:          allocation.slot.CPUs,
		UseMems:          allocation.slot.Mems,
		Stdin:            stdin,
		RootfsPath:       absRootfs,
		BindMounts:       []string{fmt.Sprintf("%s:/work", absWorkDir)},
		UseOverlayfs:     true,
	}

	resp, err := RunContext(ctx, req)
	if err != nil {
		return nil, err
	}

	report := reportFromResponse(resp, req.CPUTimeLimitUs, req.MemoryLimitBytes)
	return report, nil
}

func RunContext(ctx context.Context, req ExecRequest) (ExecResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return ExecResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "lime", "run")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(reqBytes)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ExecResponse{}, fmt.Errorf("lime run error: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var resp ExecResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return ExecResponse{}, fmt.Errorf("unmarshal response: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return resp, nil
}
