package nanopony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool[any](5, 100, 2)
	if pool == nil {
		t.Fatal("Expected worker pool to be created")
	}
	if pool.NumWorkers() < 1 {
		t.Errorf("Expected positive number of workers, got %d", pool.NumWorkers())
	}
	if pool.IsRunning() {
		t.Error("Expected pool to not be running initially")
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	pool := NewWorkerPool[any](2, 10, 2)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	if !pool.IsRunning() {
		t.Error("Expected pool to be running after Start")
	}

	for i := 0; i < 5; i++ {
		job := AcquireJob[any]()
		job.ID = "test-job"
		pool.Submit(ctx, job)
	}

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
	pool := NewWorkerPool[any](1, 1, 1)
	ctx := context.Background()

	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	job1 := AcquireJob[any]()
	job1.ID = "job1"
	pool.Submit(ctx, job1)

	job2 := AcquireJob[any]()
	job2.ID = "job2"
	pool.Submit(ctx, job2)

	job3 := AcquireJob[any]()
	job3.ID = "job3"
	err := pool.Submit(ctx, job3)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
		job3.Release()
	}

	pool.Stop()
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	pool := NewWorkerPool[any](2, 10, 2)
	ctx, cancel := context.WithCancel(context.Background())

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		atomic.AddInt32(&processed, 1)
		<-ctx.Done()
		return ctx.Err()
	})

	job1 := AcquireJob[any]()
	job1.ID = "job1"
	pool.Submit(ctx, job1)
	cancel()

	time.Sleep(100 * time.Millisecond)
	pool.Stop()
}

func TestWorkerPoolSubmitBlocking(t *testing.T) {
	pool := NewWorkerPool[any](1, 2, 1)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&processed, 1)
		return nil
	})

	done := make(chan bool, 1)
	go func() {
		job := AcquireJob[any]()
		job.ID = "blocking-job"
		err := pool.SubmitBlocking(ctx, job)
		if err != nil {
			t.Errorf("SubmitBlocking should not error when blocking, got %v", err)
			job.Release()
		}
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Error("SubmitBlocking should have completed by now")
	}

	pool.Stop()

	if processed < 1 {
		t.Errorf("Expected at least 1 job processed, got %d", processed)
	}
}

func TestWorkerPoolSubmitBlockingContextCancellation(t *testing.T) {
	pool := NewWorkerPool[any](1, 1, 1)
	ctx := context.Background()

	job1 := AcquireJob[any]()
	job1.ID = "job1"
	pool.Submit(ctx, job1)

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer timeoutCancel()

	startTime := time.Now()
	job2 := AcquireJob[any]()
	job2.ID = "should-timeout"
	err := pool.SubmitBlocking(timeoutCtx, job2)
	elapsed := time.Since(startTime)

	if err == nil {
		t.Error("SubmitBlocking should return error when context times out")
	} else {
		job2.Release()
	}

	if elapsed < 80*time.Millisecond {
		t.Errorf("SubmitBlocking should have waited ~100ms, but returned in %v", elapsed)
	}

	pool.Stop()
	job1.Release()
}

func TestWorkerPoolSubmitBlockingNoJobLoss(t *testing.T) {
	pool := NewWorkerPool[any](1, 3, 1)
	ctx := context.Background()

	processed := int32(0)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&processed, 1)
		return nil
	})

	totalJobs := 20
	for i := 0; i < totalJobs; i++ {
		job := AcquireJob[any]()
		job.ID = "job"
		err := pool.SubmitBlocking(ctx, job)
		if err != nil {
			t.Errorf("SubmitBlocking should not fail, got %v", err)
			job.Release()
		}
	}

	time.Sleep(500 * time.Millisecond)

	pool.Stop()

	if processed != int32(totalJobs) {
		t.Errorf("Expected %d jobs processed (no job loss), got %d", totalJobs, processed)
	}
}
