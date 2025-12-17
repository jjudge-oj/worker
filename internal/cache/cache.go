package cache

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/joshjms/castletown/internal/models"
	"github.com/joshjms/castletown/internal/repository"
	"github.com/joshjms/castletown/internal/store"
)

type ProblemCache struct {
	problemsRepository *repository.ProblemsRepository
	tcStore            *store.TestcaseStore

	mu       sync.Mutex
	capacity int
	ll       *list.List
	entries  map[int]*list.Element
	refCount map[int]int
	cacheDir string
}

type problemEntry struct {
	id      int
	problem *models.Problem
}

func NewProblemCache(pr *repository.ProblemsRepository, tcs *store.TestcaseStore, capacity int, cacheDir string) *ProblemCache {
	if capacity <= 0 {
		capacity = 128
	}
	if cacheDir == "" {
		cacheDir = "/var/castletown/testcases"
	}
	return &ProblemCache{
		problemsRepository: pr,
		tcStore:            tcs,
		capacity:           capacity,
		ll:                 list.New(),
		entries:            make(map[int]*list.Element),
		refCount:           make(map[int]int),
		cacheDir:           cacheDir,
	}
}

func (pc *ProblemCache) GetProblemWithLease(ctx context.Context, id int) (*models.Problem, func(), error) {
	pc.mu.Lock()
	pc.refCount[id]++
	pc.mu.Unlock()

	p, err := pc.GetProblem(ctx, id)
	if err != nil {
		pc.ReleaseProblem(id)
		return nil, nil, err
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			pc.ReleaseProblem(id)
		})
	}
	return p, release, nil
}

func (pc *ProblemCache) ReleaseProblem(id int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if count, ok := pc.refCount[id]; ok {
		if count <= 1 {
			delete(pc.refCount, id)
		} else {
			pc.refCount[id] = count - 1
		}
	}
}

func (pc *ProblemCache) GetProblem(ctx context.Context, id int) (*models.Problem, error) {
	if problem, ok := pc.getFromCache(id); ok {
		if err := pc.ensureTestcasesOnDisk(ctx, id); err != nil {
			return nil, err
		}
		return cloneProblem(problem), nil
	}

	problem, err := pc.problemsRepository.GetProblemDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := pc.ensureTestcasesOnDisk(ctx, id); err != nil {
		return nil, err
	}

	pc.addToCache(id, problem)

	return cloneProblem(problem), nil
}

func (pc *ProblemCache) GetCacheDir() string {
	return pc.cacheDir
}

func (pc *ProblemCache) getFromCache(id int) (*models.Problem, bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	elem, ok := pc.entries[id]
	if !ok {
		return nil, false
	}

	pc.ll.MoveToFront(elem)
	entry := elem.Value.(*problemEntry)
	return entry.problem, true
}

func (pc *ProblemCache) addToCache(id int, problem *models.Problem) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if elem, ok := pc.entries[id]; ok {
		pc.ll.MoveToFront(elem)
		entry := elem.Value.(*problemEntry)
		entry.problem = cloneProblem(problem)
		return
	}

	if pc.ll.Len() >= pc.capacity {
		if !pc.evictOldestLocked() {
			// Everything is in use; skip caching rather than blocking.
			return
		}
	}

	entry := &problemEntry{
		id:      id,
		problem: cloneProblem(problem),
	}
	elem := pc.ll.PushFront(entry)
	pc.entries[id] = elem
}

func (pc *ProblemCache) evictOldestLocked() bool {
	for elem := pc.ll.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*problemEntry)
		if pc.refCount[entry.id] > 0 {
			continue
		}
		pc.ll.Remove(elem)
		delete(pc.entries, entry.id)
		_ = os.RemoveAll(filepath.Join(pc.cacheDir, fmt.Sprintf("%d", entry.id)))
		return true
	}
	return false
}

func (pc *ProblemCache) ensureTestcasesOnDisk(ctx context.Context, id int) error {
	finalDir := filepath.Join(pc.cacheDir, fmt.Sprintf("%d", id))

	info, err := os.Stat(finalDir)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		if remErr := os.RemoveAll(finalDir); remErr != nil {
			return remErr
		}
	}

	if pc.tcStore != nil {
		if err := pc.retrieveTestcasesFromObjectStore(id); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return fmt.Errorf("testcases for problem %d not found", id)
}

func (pc *ProblemCache) retrieveTestcasesFromObjectStore(id int) error {
	r, err := pc.tcStore.Get(context.Background(), "testcases", fmt.Sprintf("%d.tar.gz", id))
	if err != nil {
		return err
	}
	defer r.Close()

	if err := untarGzReader(r, pc.cacheDir, id); err != nil {
		_ = os.RemoveAll(filepath.Join(pc.cacheDir, fmt.Sprintf("%d", id)))
		return err
	}

	return nil
}

func cloneProblem(p *models.Problem) *models.Problem {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}
