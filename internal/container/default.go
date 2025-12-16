package container

func UseDefaultConfig() *Config {
	return &Config{
		Env:              []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		Cwd:              "/work",
		TimeLimitUs:      1000000,
		MemoryLimitBytes: 256 * 1024 * 1024,
		PidLimit:         64,
		UseThreads:       1,
		CpusetCPUs:       "0",
		CpusetMems:       "0",
		Rlimit: &RlimitConfig{
			Core: &Rlimit{
				Hard: 0,
				Soft: 0,
			},
			Fsize: &Rlimit{
				Hard: 1 * 1024 * 1024,
				Soft: 1 * 1024 * 1024,
			},
			NoFile: &Rlimit{
				Hard: 64,
				Soft: 64,
			},
		},
	}
}
