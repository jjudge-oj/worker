package container

import (
	"context"
	"sync/atomic"
)

const StartUIDGID = 1000000
const DefaultSize = 65536

type Slot struct {
	UIDStart uint32
	UIDSize  uint32
	GIDStart uint32
	GIDSize  uint32
	CPU      int
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
		sp.ch = make(chan Slot, 1)
		sp.ch <- Slot{
			UIDStart: StartUIDGID,
			UIDSize:  DefaultSize,
			GIDStart: StartUIDGID,
			GIDSize:  DefaultSize,
			CPU:      0,
		}
	}

	return sp
}

func WithMaxConcurrency(n int) SlotPoolOption {
	return func(sp *SlotPool) {
		if n < 1 {
			n = 1
		}

		sp.ch = make(chan Slot, n)
		for i := 0; i < n; i++ {
			sp.ch <- Slot{
				UIDStart: StartUIDGID + uint32(i)*DefaultSize,
				UIDSize:  DefaultSize,
				GIDStart: StartUIDGID + uint32(i)*DefaultSize,
				GIDSize:  DefaultSize,
				CPU:      i,
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
