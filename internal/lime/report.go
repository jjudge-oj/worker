package lime

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
	Signal   int
	Stdout   string
	Stderr   string
	CPUTime  uint64
	Memory   uint64
	WallTime uint64
}

func reportFromResponse(resp ExecResponse, timeLimitUs, memoryLimitBytes uint64) *Report {
	status := STATUS_OK
	if timeLimitUs > 0 && (resp.CPUTimeUs > timeLimitUs || resp.WallTimeUs > timeLimitUs) {
		status = STATUS_TIME_LIMIT_EXCEEDED
	} else if memoryLimitBytes > 0 && resp.MemoryBytes > memoryLimitBytes {
		status = STATUS_MEMORY_LIMIT_EXCEEDED
	} else if resp.ExitCode != 0 || resp.TermSignal != 0 {
		status = STATUS_RUNTIME_ERROR
	}

	return &Report{
		Status:   status,
		ExitCode: resp.ExitCode,
		Signal:   resp.TermSignal,
		Stdout:   resp.Stdout,
		Stderr:   resp.Stderr,
		CPUTime:  resp.CPUTimeUs,
		Memory:   resp.MemoryBytes,
		WallTime: resp.WallTimeUs,
	}
}
