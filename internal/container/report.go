package container

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

type Status string

const (
	STATUS_OK                    Status = "OK"
	STATUS_RUNTIME_ERROR         Status = "RUNTIME_ERROR"
	STATUS_TIME_LIMIT_EXCEEDED   Status = "TIME_LIMIT_EXCEEDED"
	STATUS_MEMORY_LIMIT_EXCEEDED Status = "MEMORY_LIMIT_EXCEEDED"
	STATUS_OUTPUT_LIMIT_EXCEEDED Status = "OUTPUT_LIMIT_EXCEEDED"
	STATUS_TERMINATED            Status = "TERMINATED"
	STATUS_UNKNOWN               Status = "UNKNOWN"
	STATUS_SKIPPED               Status = "SKIPPED"
)

type Report struct {
	Status   Status
	ExitCode int
	Signal   syscall.Signal
	Stdout   string
	Stderr   string
	CPUTime  uint64
	Memory   uint64
	WallTime int64

	StartAt  time.Time
	FinishAt time.Time
}

func (r Report) String() string {
	stdoutTrim := r.Stdout
	if len(stdoutTrim) > 200 {
		stdoutTrim = stdoutTrim[:200]
	}

	stderrTrim := r.Stderr
	if len(stderrTrim) > 200 {
		stderrTrim = stderrTrim[:200]
	}

	return fmt.Sprintf("status: %s\nexit code: %d\nsignal: %d\nstdout: %s\nstderr:%s\ncpu:%d usec\nmemory:%d bytes\n", r.Status, r.ExitCode, r.Signal, stdoutTrim, stderrTrim, r.CPUTime, r.Memory)
}

func (c *Container) makeReport(stdoutBuf, stderrBuf io.Reader, state *os.ProcessState, timeLimitExceeded bool, startAt, finishAt time.Time) (*Report, error) {
	stdout, err := io.ReadAll(stdoutBuf)
	if err != nil {
		return nil, fmt.Errorf("error reading stdout: %w", err)
	}

	stderr, err := io.ReadAll(stderrBuf)
	if err != nil {
		return nil, fmt.Errorf("error reading stderr: %w", err)
	}

	cgManager, err := loadCgroup(c.id)
	if err != nil {
		return nil, fmt.Errorf("error loading cgroup: %w", err)
	}

	stats, err := cgManager.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting cgroup stats: %w", err)
	}

	var status Status

	cpuUsageUs := stats.GetCPU().GetUsageUsec()
	memoryMaxUsage := stats.GetMemory().GetMaxUsage()

	switch {
	case timeLimitExceeded || cpuUsageUs > uint64(c.config.TimeLimitUs):
		status = STATUS_TIME_LIMIT_EXCEEDED
	case memoryMaxUsage > uint64(c.config.MemoryLimitBytes):
		status = STATUS_MEMORY_LIMIT_EXCEEDED
	case state.ExitCode() != 0:
		status = STATUS_RUNTIME_ERROR
	default:
		status = STATUS_OK
	}

	return &Report{
		Status:   status,
		ExitCode: state.ExitCode(),
		Signal:   state.Sys().(syscall.WaitStatus).Signal(),
		Stdout:   string(stdout),
		Stderr:   string(stderr),
		CPUTime:  cpuUsageUs,
		Memory:   memoryMaxUsage,
		StartAt:  startAt,
		FinishAt: finishAt,
	}, nil
}
