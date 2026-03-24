package nanopony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(5, 100)
	if pool == nil {
		t.Fatal("Expected worker pool to be created")
	}
	if pool.numWorkers != 5 {
		t.Errorf("Expected 5 workers, got %d", pool.numWorkers)
	}
	if pool.running {
		t.Error("Expected pool to not be running initially")
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	if !pool.IsRunning() {
		t.Error("Expected pool to be running after Start")
	}

	// Submit jobs
	for i := 0; i < 5; i++ {
		pool.Submit(ctx, Job{ID: "test-job"})
	}

	// Give workers time to process
	time.Sleep(100 * time.Millisecond)

	pool.Stop()

	if pool.IsRunning() {
		t.Error("Expected pool to be stopped after Stop")
	}

	if processed != 5 {
		t.Errorf("Expected 5 jobs processed, got %d", processed)
	}
}

func TestWorkerPoolSubmitQueueFull(t *testing.T) {
	pool := NewWorkerPool(1, 2)
	ctx := context.Background()

	// Block the worker
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	// Fill the queue
	pool.Submit(ctx, Job{ID: "job1"})
	pool.Submit(ctx, Job{ID: "job2"})

	// This should fail with queue full
	err := pool.Submit(ctx, Job{ID: "job3"})
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}

	pool.Stop()
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	ctx, cancel := context.WithCancel(context.Background())

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		atomic.AddInt32(&processed, 1)
		<-ctx.Done()
		return ctx.Err()
	})

	pool.Submit(ctx, Job{ID: "job1"})
	cancel()

	time.Sleep(100 * time.Millisecond)
	pool.Stop()
}
