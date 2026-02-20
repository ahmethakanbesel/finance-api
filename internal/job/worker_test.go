package job

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type mockProcessor struct {
	processed atomic.Int64
}

func (m *mockProcessor) Process(_ context.Context, _ *Job) error {
	m.processed.Add(1)
	return nil
}

func TestWorkerPool_ProcessesPendingJobs(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	// Seed pending jobs
	for i := 0; i < 3; i++ {
		_ = repo.Create(ctx, &Job{Source: "tefas", Symbol: "YAC", Status: StatusPending})
	}

	proc := &mockProcessor{}
	pool := NewWorkerPool(repo, proc, 2)
	pool.pollInterval = 50 * time.Millisecond

	poolCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		pool.Run(poolCtx)
		close(done)
	}()

	// Notify to kick off processing
	pool.Notify()

	// Wait for all jobs to be processed
	deadline := time.After(2 * time.Second)
	for proc.processed.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for jobs to be processed, got %d", proc.processed.Load())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()
	<-done
}

func TestWorkerPool_NotifyWakesWorker(t *testing.T) {
	repo := newMockRepo()
	proc := &mockProcessor{}
	pool := NewWorkerPool(repo, proc, 1)
	pool.pollInterval = 10 * time.Second // long poll so only Notify wakes it

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		pool.Run(ctx)
		close(done)
	}()

	// Create a pending job after pool started
	_ = repo.Create(context.Background(), &Job{Source: "tefas", Symbol: "YAC", Status: StatusPending})
	pool.Notify()

	deadline := time.After(2 * time.Second)
	for proc.processed.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("timed out: Notify did not wake worker")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()
	<-done
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	repo := newMockRepo()
	proc := &mockProcessor{}
	pool := NewWorkerPool(repo, proc, 2)
	pool.pollInterval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		pool.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// OK — workers drained
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown")
	}
}
