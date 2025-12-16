package container

import (
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"

	_ "github.com/opencontainers/cgroups/devices"
)

func createSpec(containerID string, cfg *Config) (*specs.Spec, error) {
	slicePath, err := getSlicePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get slice path: %w", err)
	}

	mounts := getMounts(cfg)

	spec := &specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			NoNewPrivileges: true,
		},
		// Use a dummy rootfs; actual rootfs will be mounted via overlayfs
		Root: &specs.Root{
			Path:     cfg.RootfsImageDir,
			Readonly: false,
		},
		Hostname: "castletown",
		Mounts:   mounts,
		Linux: &specs.Linux{
			CgroupsPath: filepath.Join(slicePath, fmt.Sprintf("castletown-%s.scope", containerID), containerID),
			Resources:   cgroupResources(cfg),
			UIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: cfg.UserNamespace.UID.ContainerID,
					HostID:      cfg.UserNamespace.UID.HostID,
					Size:        cfg.UserNamespace.UID.Size,
				},
			},
			GIDMappings: []specs.LinuxIDMapping{
				{
					HostID:      cfg.UserNamespace.GID.HostID,
					ContainerID: cfg.UserNamespace.GID.ContainerID,
					Size:        cfg.UserNamespace.GID.Size,
				},
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.CgroupNamespace},
				{Type: specs.PIDNamespace},
				{Type: specs.IPCNamespace},
				{Type: specs.UTSNamespace},
				{Type: specs.MountNamespace},
				{Type: specs.UserNamespace},
				{Type: specs.NetworkNamespace},
			},
			// https://github.com/moby/moby/blob/master/oci/defaults.go
			MaskedPaths: []string{
				"/proc/asound",
				"/proc/acpi",
				"/proc/interrupts", // https://github.com/moby/moby/security/advisories/GHSA-6fw5-f8r9-fgfm
				"/proc/kcore",
				"/proc/keys",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/proc/scsi",
				"/sys/firmware",
				"/sys/devices/virtual/powercap", // https://github.com/moby/moby/security/advisories/GHSA-jq35-85cj-fj4p
			},
			ReadonlyPaths: []string{
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
	}

	return spec, nil
}

func getMounts(cfg *Config) []specs.Mount {
	mounts := make([]specs.Mount, 0)

	rootfsMount := specs.Mount{
		Destination: "/",
		Type:        "overlay",
		Source:      "overlay",
		Options: []string{
			"rw",
			"userxattr",
			"xino=off",
			"index=off",
			fmt.Sprintf("upperdir=%s", cfg.Overlay.UpperDir),
			fmt.Sprintf("lowerdir=%s", cfg.Overlay.LowerDir),
			fmt.Sprintf("workdir=%s", cfg.Overlay.WorkDir),
		},
	}

	mounts = append(mounts, rootfsMount)

	bindMount := specs.Mount{
		Destination: "/work",
		Type:        "bind",
		Source:      cfg.BindMount,
		Options: []string{
			"rbind",
			"rw",
			"exec",
			"nosuid",
			"nodev",
			"ridmap",
		},
		UIDMappings: []specs.LinuxIDMapping{
			{
				ContainerID: 0,
				HostID:      cfg.UserNamespace.UID.HostID,
				Size:        1,
			},
		},
		GIDMappings: []specs.LinuxIDMapping{
			{
				ContainerID: 0,
				HostID:      cfg.UserNamespace.GID.HostID,
				Size:        1,
			},
		},
	}

	mounts = append(mounts, bindMount)

	mounts = append(mounts, defaultMounts()...)

	return mounts
}

func defaultMounts() []specs.Mount {
	return []specs.Mount{
		{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
		},
		{
			Destination: "/dev",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options: []string{
				"nosuid",
				"strictatime",
				"mode=755",
				"size=65536k",
			},
		},
		{
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Source:      "shm",
			Options: []string{
				"nosuid",
				"noexec",
				"nodev",
				"mode=1777",
				"size=65536k",
			},
		},
		{
			Destination: "/tmp",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options: []string{
				"nosuid",
				"noexec",
				"nodev",
				"size=128m",
				"nr_inodes=4k",
			},
		},
		// {
		// 	Destination: "/sys",
		// 	Type:        "sysfs",
		// 	Source:      "sysfs",
		// 	Options:     []string{"nosuid", "noexec", "nodev", "ro"},
		// },
	}
}

func cgroupResources(cfg *Config) *specs.LinuxResources {
	resources := &specs.LinuxResources{
		CPU:    &specs.LinuxCPU{},
		Memory: &specs.LinuxMemory{},
		Pids:   &specs.LinuxPids{},
	}

	var cpuPeriod uint64 = 100000
	var cpuQuota int64 = 100000 * cfg.UseThreads

	resources.CPU.Quota = &cpuQuota
	resources.CPU.Period = &cpuPeriod

	if cfg.CpusetCPUs != "" {
		resources.CPU.Cpus = cfg.CpusetCPUs
	}

	if cfg.CpusetMems != "" {
		resources.CPU.Mems = cfg.CpusetMems
	}

	if cfg.MemoryLimitBytes != 0 {
		limit := cfg.MemoryLimitBytes
		resources.Memory.Limit = &limit
	}

	if cfg.PidLimit != 0 {
		limit := cfg.PidLimit
		resources.Pids.Limit = limit
	}

	return resources
}
