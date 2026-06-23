package nanopony

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(5, 100, 2)
	if pool == nil {
		t.Fatal("Expected worker pool to be created")
	}
	// Sharded pool approximates workers per shard * numShards
	if pool.NumWorkers() < 1 {
		t.Errorf("Expected positive number of workers, got %d", pool.NumWorkers())
	}
	if pool.IsRunning() {
		t.Error("Expected pool to not be running initially")
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	pool := NewWorkerPool(2, 10, 2)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	// Test idempotent Start
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
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

	// Test idempotent Stop
	pool.Stop()

	if pool.IsRunning() {
		t.Error("Expected pool to be stopped after Stop")
	}

	if processed != 5 {
		t.Errorf("Expected 5 jobs processed, got %d", processed)
	}
}

func TestWorkerPoolErrorReporting(t *testing.T) {
	pool := NewWorkerPool(1, 10, 1)
	ctx := context.Background()

	errLogged := make(chan error, 1)

	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		return context.DeadlineExceeded // Return an error to trigger reportError
	})

	// Catch error from shard's errChan
	go func() {
		err := <-pool.shards[0].errChan
		errLogged <- err
	}()

	job := AcquireJob()
	job.ID = "error-job"
	pool.Submit(ctx, job)

	select {
	case err := <-errLogged:
		if err == nil {
			t.Error("Expected error to be reported")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Timed out waiting for error report")
	}

	pool.Stop()
}

func TestWorkerPoolNewExtremeParams(t *testing.T) {
	// numWorkers < 1
	p1 := NewWorkerPool(0, 10, 1)
	if p1.totalWorkers != 1 {
		t.Errorf("Expected 1 worker, got %d", p1.totalWorkers)
	}

	// queueSize < 1
	p2 := NewWorkerPool(1, 0, 1)
	if cap(p2.shards[0].jobChan) != 1 {
		t.Errorf("Expected queue size 1, got %d", cap(p2.shards[0].jobChan))
	}

	// numShards < 1
	p3 := NewWorkerPool(1, 10, 0)
	if len(p3.shards) != 1 {
		t.Errorf("Expected 1 shard, got %d", len(p3.shards))
	}

	// numShards > numWorkers
	p4 := NewWorkerPool(2, 10, 5)
	if len(p4.shards) != 2 {
		t.Errorf("Expected 2 shards, got %d", len(p4.shards))
	}
}

func TestWorkerPoolSubmitToStoppedPool(t *testing.T) {
	pool := NewWorkerPool(1, 1, 1)
	pool.Start(context.Background(), func(ctx context.Context, job *Job) error { return nil })
	pool.Stop() // Ensure it's stopped

	job := AcquireJob()
	job.ID = "stopped-job"
	err := pool.Submit(context.Background(), job)
	if err != ErrPoolStopped {
		t.Errorf("Expected ErrPoolStopped, got %v", err)
	}
	job.Release()

	job2 := AcquireJob()
	job2.ID = "stopped-job-2"
	err = pool.SubmitBlocking(context.Background(), job2)
	if err != ErrPoolStopped {
		t.Errorf("Expected ErrPoolStopped, got %v", err)
	}
	job2.Release()
}

func TestWorkerPoolNumShards(t *testing.T) {
	pool := NewWorkerPool(4, 10, 2)
	if pool.NumShards() != 2 {
		t.Errorf("Expected 2 shards, got %d", pool.NumShards())
	}
}

func TestWorkerPoolDrainOnCancel(t *testing.T) {
	pool := NewWorkerPool(1, 10, 1)
	ctx, cancel := context.WithCancel(context.Background())

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	// Fill queue
	for i := 0; i < 5; i++ {
		job := AcquireJob()
		job.ID = "drain-job"
		pool.Submit(ctx, job)
	}

	cancel() // Cancel context to trigger drain logic
	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	// Drain logic should ensure jobs are released
}

func TestWorkerPoolSubmitQueueFull(t *testing.T) {
	// Small queue per shard. With 1 worker and 1 shard, queue size 1.
	pool := NewWorkerPool(1, 1, 1)
	ctx := context.Background()

	workerStarted := make(chan struct{})
	var once sync.Once

	// Block the worker
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		once.Do(func() {
			close(workerStarted)
		})
		time.Sleep(1 * time.Second)
		return nil
	})

	// Fill the queue
	// First job is picked by worker, but worker sleeps
	job1 := AcquireJob()
	job1.ID = "job1"
	if err := pool.Submit(ctx, job1); err != nil {
		t.Fatalf("Failed to submit job1: %v", err)
	}

	// Wait for worker to actually pick up job1
	select {
	case <-workerStarted:
		// Worker started processing job1
	case <-time.After(2 * time.Second):
		t.Fatal("Worker never started processing job1")
	}

	// Second job fills the queue (size 1)
	job2 := AcquireJob()
	job2.ID = "job2"
	if err := pool.Submit(ctx, job2); err != nil {
		t.Errorf("Expected job2 to be submitted, got %v", err)
		job2.Release()
	}

	// This should fail with queue full
	job3 := AcquireJob()
	job3.ID = "job3"
	err := pool.Submit(ctx, job3)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
	job3.Release()

	pool.Stop()
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	pool := NewWorkerPool(2, 10, 2)
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
	pool := NewWorkerPool(1, 2, 1)
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
	pool := NewWorkerPool(1, 1, 1) // Very small queue
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
	pool := NewWorkerPool(1, 3, 1)
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
