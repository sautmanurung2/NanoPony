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
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	if !pool.IsRunning() {
		t.Error("Expected pool to be running after Start")
	}

	// Submit jobs
	for i := 0; i < 5; i++ {
		job := AcquireJob()
		job.ID = "test-job"
		pool.Submit(ctx, job)
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
	pool := NewWorkerPool(1, 1) // Small queue
	ctx := context.Background()

	// Block the worker
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	// Fill the queue
	job1 := AcquireJob()
	job1.ID = "job1"
	pool.Submit(ctx, job1)

	job2 := AcquireJob()
	job2.ID = "job2"
	pool.Submit(ctx, job2)

	// This should fail with queue full
	job3 := AcquireJob()
	job3.ID = "job3"
	err := pool.Submit(ctx, job3)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
		job3.Release()
	}

	pool.Stop()
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	ctx, cancel := context.WithCancel(context.Background())

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processed, 1)
		<-ctx.Done()
		return ctx.Err()
	})

	job1 := AcquireJob()
	job1.ID = "job1"
	pool.Submit(ctx, job1)
	cancel()

	time.Sleep(100 * time.Millisecond)
	pool.Stop()
}

func TestWorkerPoolSubmitBlocking(t *testing.T) {
	pool := NewWorkerPool(1, 2)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		time.Sleep(50 * time.Millisecond) // Slow worker
		atomic.AddInt32(&processed, 1)
		return nil
	})

	// SubmitBlocking should block when queue is full, not return error
	done := make(chan bool, 1)
	go func() {
		// This should block until worker processes jobs
		job := AcquireJob()
		job.ID = "blocking-job"
		err := pool.SubmitBlocking(ctx, job)
		if err != nil {
			t.Errorf("SubmitBlocking should not error when blocking, got %v", err)
			job.Release()
		}
		done <- true
	}()

	// Give it time to block
	time.Sleep(100 * time.Millisecond)

	// SubmitBlocking should still be waiting (worker is slow)
	select {
	case <-done:
		// Good, it completed
	case <-time.After(50 * time.Millisecond):
		t.Error("SubmitBlocking should have completed by now")
	}

	pool.Stop()

	if processed < 1 {
		t.Errorf("Expected at least 1 job processed, got %d", processed)
	}
}

func TestWorkerPoolSubmitBlockingContextCancellation(t *testing.T) {
	pool := NewWorkerPool(1, 1) // Very small queue
	ctx := context.Background()

	// Don't start the pool - no workers processing
	// This ensures queue will fill up and block

	// Fill the queue completely (size=1)
	job1 := AcquireJob()
	job1.ID = "job1"
	pool.Submit(ctx, job1)

	// Now SubmitBlocking with timeout should block (queue is full, no workers)
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer timeoutCancel()

	startTime := time.Now()
	job2 := AcquireJob()
	job2.ID = "should-timeout"
	err := pool.SubmitBlocking(timeoutCtx, job2)
	elapsed := time.Since(startTime)

	// Should timeout because queue is full and no worker to process
	if err == nil {
		t.Error("SubmitBlocking should return error when context times out")
	} else {
		job2.Release()
	}

	if elapsed < 80*time.Millisecond {
		t.Errorf("SubmitBlocking should have waited ~100ms, but returned in %v", elapsed)
	}

	pool.Stop()
	job1.Release() // Release job1 manually because no worker processed it
}

func TestWorkerPoolSubmitBlockingNoJobLoss(t *testing.T) {
	pool := NewWorkerPool(1, 3)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&processed, 1)
		return nil
	})

	// Submit many jobs using SubmitBlocking
	totalJobs := 20
	for i := 0; i < totalJobs; i++ {
		job := AcquireJob()
		job.ID = "job"
		err := pool.SubmitBlocking(ctx, job)
		if err != nil {
			t.Errorf("SubmitBlocking should not fail, got %v", err)
			job.Release()
		}
	}

	// Wait for all jobs to be processed
	time.Sleep(500 * time.Millisecond)

	pool.Stop()

	// ALL jobs should be processed (no job loss!)
	if processed != int32(totalJobs) {
		t.Errorf("Expected %d jobs processed (no job loss), got %d", totalJobs, processed)
	}
}
