package container

import (
	"fmt"
	"io"
)

// Config holds the configuration for the sandboxed process and environment.
type Config struct {
	// Root filesystem image directory. Will be used as the base for the overlay filesystem.
	RootfsImageDir string

	// Command-line arguments to pass to the sandboxed process.
	Args []string
	// Standard input content for the sandboxed process.
	Stdin io.Reader
	// Working directory inside the sandbox.
	Cwd string
	// Environment variables for the sandboxed process.
	Env []string

	// User namespace configuration.
	UserNamespace *UserNamespaceConfig

	// CPU time limit in microseconds.
	TimeLimitUs int64
	// Memory limit in bytes.
	MemoryLimitBytes int64
	// Maximum number of PIDs allowed.
	PidLimit int64
	// Number of threads to use.
	UseThreads int64
	// CPU core(s) to which the sandboxed process is pinned to.
	CpusetCPUs string
	// CPU memory node(s) to which the sandboxed process is pinned to.
	CpusetMems string

	// Resource limits configuration.
	Rlimit *RlimitConfig

	// Directory on the host to be bind-mounted into the sandbox at /work.
	BindMount string

	// Overlay filesystem configuration.
	Overlay *Overlay

	allocation *Allocation
}

// UseAllocation configures the sandbox config with the given allocation.
func (c *Config) UseAllocation(a *Allocation) {
	c.allocation = a

	c.UserNamespace = &UserNamespaceConfig{
		UID: IDMapping{
			ContainerID: 0,
			HostID:      a.slot.UIDStart,
			Size:        a.slot.UIDSize,
		},
		GID: IDMapping{
			ContainerID: 0,
			HostID:      a.slot.GIDStart,
			Size:        a.slot.GIDSize,
		},
	}

	c.CpusetCPUs = fmt.Sprintf("%d", a.slot.CPU)
	c.CpusetMems = "0"
}

type UserNamespaceConfig struct {
	UID IDMapping
	GID IDMapping
}

type IDMapping struct {
	ContainerID uint32
	HostID      uint32
	Size        uint32
}

type RlimitConfig struct {
	Core   *Rlimit
	Fsize  *Rlimit
	NoFile *Rlimit
}

type Rlimit struct {
	Hard uint64
	Soft uint64
}
