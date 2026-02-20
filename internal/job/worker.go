package job

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Processor handles execution of a claimed job.
type Processor interface {
	Process(ctx context.Context, j *Job) error
}

// WorkerPool runs a fixed number of goroutines that claim and process pending jobs.
type WorkerPool struct {
	repo         Repository
	processor    Processor
	workers      int
	notify       chan struct{}
	pollInterval time.Duration
}

// NewWorkerPool creates a pool with the given number of workers.
func NewWorkerPool(repo Repository, processor Processor, workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}
	return &WorkerPool{
		repo:         repo,
		processor:    processor,
		workers:      workers,
		notify:       make(chan struct{}, 1),
		pollInterval: 5 * time.Second,
	}
}

// Notify wakes idle workers to check for pending jobs. Non-blocking.
func (wp *WorkerPool) Notify() {
	select {
	case wp.notify <- struct{}{}:
	default:
	}
}

// Run starts worker goroutines and blocks until ctx is cancelled and all
// workers have drained.
func (wp *WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := range wp.workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			wp.loop(ctx, id)
		}(i)
	}
	wg.Wait()
}

func (wp *WorkerPool) loop(ctx context.Context, id int) {
	ticker := time.NewTicker(wp.pollInterval)
	defer ticker.Stop()

	for {
		// Drain all available pending jobs before waiting.
		wp.drain(ctx, id)

		select {
		case <-ctx.Done():
			return
		case <-wp.notify:
		case <-ticker.C:
		}
	}
}

func (wp *WorkerPool) drain(ctx context.Context, id int) {
	for {
		if ctx.Err() != nil {
			return
		}

		j, err := wp.repo.ClaimPending(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			slog.Error("worker: claim pending", "worker", id, "error", err)
			return
		}
		if j == nil {
			return // no more pending jobs
		}

		slog.Info("worker: processing job", "worker", id, "job", j.ID, "source", j.Source, "symbol", j.Symbol)

		if err := wp.processor.Process(ctx, j); err != nil {
			slog.Error("worker: process job", "worker", id, "job", j.ID, "error", err)
		}
	}
}
