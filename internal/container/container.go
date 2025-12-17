package container

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshjms/castletown/internal/config"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"golang.org/x/sys/unix"
)

type Container struct {
	id            string
	config        *Config
	runtimeCfg    *config.Config
	containerImpl *libcontainer.Container
}

func NewContainer(id string, cfg *config.Config, opts ...ContainerOptions) *Container {
	if id == "" {
		id = uuid.NewString()
	}
	c := &Container{
		id:         id,
		runtimeCfg: cfg,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type ContainerOptions func(*Container)

func WithContainerConfig(cfg *Config) ContainerOptions {
	return func(c *Container) {
		c.config = cfg
	}
}

func (c *Container) Init(ctx context.Context) error {
	spec, err := createSpec(c.id, c.config)
	if err != nil {
		return fmt.Errorf("error creating spec: %w", err)
	}

	libcontainerConfig, err := specconv.CreateLibcontainerConfig(&specconv.CreateOpts{
		UseSystemdCgroup: false,
		Spec:             spec,
	})
	if err != nil {
		return fmt.Errorf("error creating libcontainer config: %w", err)
	}

	container, err := libcontainer.Create(c.runtimeCfg.Judge.LibcontainerDir, c.id, libcontainerConfig)
	if err != nil {
		return fmt.Errorf("error creating container: %w", err)
	}

	c.containerImpl = container

	return nil
}

// Run runs a command inside the container and returns a Report
func (c *Container) Run(ctx context.Context) (*Report, error) {
	noNewPrivileges := true

	var stdoutBuf, stderrBuf bytes.Buffer

	rlimits := getRlimits(c.config)

	process := &libcontainer.Process{
		Args:            c.config.Args,
		Env:             c.config.Env,
		UID:             0,
		GID:             0,
		Cwd:             c.config.Cwd,
		NoNewPrivileges: &noNewPrivileges,
		Stdin:           c.config.Stdin,
		Stdout:          &stdoutBuf,
		Stderr:          &stderrBuf,
		Rlimits:         rlimits,
		Init:            true,
	}

	startAt := time.Now()

	if err := c.containerImpl.Run(process); err != nil {
		return nil, fmt.Errorf("error running container: %w", err)
	}

	processFinished := make(chan interface{}, 1)
	timeLimitExceeded := false

	go func() {
		select {
		case <-processFinished:
		case <-time.After(time.Duration(c.config.TimeLimitUs) * time.Microsecond * 3):
			timeLimitExceeded = true
			c.containerImpl.Signal(unix.SIGKILL)
		}
	}()

	state, _ := process.Wait()
	processFinished <- struct{}{}

	finishAt := time.Now()

	return c.makeReport(&stdoutBuf, &stderrBuf, state, timeLimitExceeded, startAt, finishAt)
}

func (c *Container) Destroy() error {
	if c.containerImpl != nil {
		c.containerImpl.Destroy()
	}

	if c.config.Overlay != nil {
		if err := Cleanup(c.config.Overlay); err != nil {
			return fmt.Errorf("error cleaning up overlay: %w", err)
		}
	}

	if c.config.allocation != nil {
		c.config.allocation.Release()
	}

	return nil
}

func getRlimits(cfg *Config) []configs.Rlimit {
	if cfg.Rlimit == nil {
		return nil
	}

	var rlimits []configs.Rlimit

	if cfg.Rlimit.Core != nil {
		rlimits = append(rlimits, configs.Rlimit{
			Type: unix.RLIMIT_CORE,
			Hard: cfg.Rlimit.Core.Hard,
			Soft: cfg.Rlimit.Core.Soft,
		})
	}

	if cfg.Rlimit.Fsize != nil {
		rlimits = append(rlimits, configs.Rlimit{
			Type: unix.RLIMIT_FSIZE,
			Hard: cfg.Rlimit.Fsize.Hard,
			Soft: cfg.Rlimit.Fsize.Soft,
		})
	}

	if cfg.Rlimit.NoFile != nil {
		rlimits = append(rlimits, configs.Rlimit{
			Type: unix.RLIMIT_NOFILE,
			Hard: cfg.Rlimit.NoFile.Hard,
			Soft: cfg.Rlimit.NoFile.Soft,
		})
	}

	return rlimits
}
