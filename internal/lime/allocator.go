package lime

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

type Slot struct {
	CPUs string
	Mems string
}

type SlotPool struct {
	ch chan Slot
}

type SlotPoolOption func(*SlotPool)

func NewSlotPool(opts ...SlotPoolOption) *SlotPool {
	sp := &SlotPool{}
	for _, opt := range opts {
		opt(sp)
	}

	if sp.ch == nil {
		WithMaxConcurrency(1)(sp)
	}

	return sp
}

func WithMaxConcurrency(n int) SlotPoolOption {
	return func(sp *SlotPool) {
		if n < 1 {
			n = 1
		}

		cpusEnv := strings.TrimSpace(os.Getenv("CASTLETOWN_CPUS"))
		memsEnv := strings.TrimSpace(os.Getenv("CASTLETOWN_MEMS"))

		cpus := parseCPUSet(cpusEnv)
		mems := parseCPUSet(memsEnv)
		if len(cpus) == 0 {
			cpus = []string{"0"}
		}
		if len(mems) == 0 {
			mems = []string{"0"}
		}

		if cpusEnv != "" && len(cpus) < n {
			n = len(cpus)
			if n < 1 {
				n = 1
			}
		}

		sp.ch = make(chan Slot, n)
		for i := 0; i < n; i++ {
			sp.ch <- Slot{
				CPUs: cpus[i%len(cpus)],
				Mems: mems[i%len(mems)],
			}
		}
	}
}

type Allocation struct {
	pool     *SlotPool
	slot     Slot
	released atomic.Bool
}

func (sp *SlotPool) Allocate(ctx context.Context) (*Allocation, error) {
	select {
	case r := <-sp.ch:
		return &Allocation{pool: sp, slot: r}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (a *Allocation) Release() {
	if a == nil || a.pool == nil {
		return
	}
	if !a.released.CompareAndSwap(false, true) {
		return
	}
	a.pool.ch <- a.slot
}

func parseCPUSet(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return nil
			}
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil
			}
			if start > end {
				return nil
			}
			for i := start; i <= end; i++ {
				out = append(out, strconv.Itoa(i))
			}
			continue
		}

		if _, err := strconv.Atoi(part); err != nil {
			return nil
		}
		out = append(out, part)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
