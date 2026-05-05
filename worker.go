// worker.go — Concurrent job processing using a fan-out worker pool.
//
// Architecture (fan-out pattern):
//
//	                   ┌─── worker 0 ──→ handler(job)
//	Submit(job) ──→ jobChan ──┼─── worker 1 ──→ handler(job)
//	                   └─── worker N ──→ handler(job)
//
// Submit() is non-blocking (returns ErrQueueFull if buffer full).
// SubmitBlocking() waits for space in the channel (backpressure).
// Workers run as goroutines, each pulling from the shared jobChan.
package nanopony

import (
	"context"
	"fmt"
	"sync"
)

// WorkerPool manages a collection of concurrent workers that process jobs from a shared queue.
// It provides backpressure mechanisms via blocking and non-blocking job submission.
type WorkerPool struct {
	numWorkers int
	jobChan    chan Job
	errChan    chan error
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	handler    JobHandler
	mu         sync.RWMutex
	running    bool
	stopOnce   sync.Once
}

// NewWorkerPool creates a new WorkerPool ready to be started.
//
// Parameters:
//   - numWorkers: How many goroutines to run in parallel.
//   - queueSize: The size of the job buffer (channel capacity).
func NewWorkerPool(numWorkers, queueSize int) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobChan:    make(chan Job, queueSize),
		errChan:    make(chan error, queueSize),
	}
}

// Start activates the worker pool and spawns the background worker goroutines.
// The provided context controls the lifecycle of the workers.
func (wp *WorkerPool) Start(ctx context.Context, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}

	wp.ctx, wp.cancel = context.WithCancel(ctx)
	wp.handler = handler
	wp.running = true

	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(wp.ctx, i)
	}
}

// worker is the internal execution loop for a single goroutine.
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				return
			}

			if err := wp.handler(ctx, job); err != nil {
				wp.reportError(fmt.Errorf("worker %d: job %s failed: %w", id, job.ID, err))
			}
		}
	}
}

// reportError safely sends an error to the error channel or logs it if the channel is full.
func (wp *WorkerPool) reportError(err error) {
	select {
	case wp.errChan <- err:
	default:
		LogFramework("WARNING", "WorkerPool", fmt.Sprintf("Error channel full, discarded: %v", err))
	}
}

// Submit sends a job to the queue. Returns ErrQueueFull if the buffer is at capacity.
func (wp *WorkerPool) Submit(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.jobChan <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitBlocking sends a job to the queue and waits until space is available.
// This is the recommended way to handle backpressure in high-load scenarios.
func (wp *WorkerPool) SubmitBlocking(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.jobChan <- job:
		return nil
	}
}

// Stop gracefully shuts down the worker pool, waiting for all workers to finish current jobs.
func (wp *WorkerPool) Stop() {
	wp.stopOnce.Do(func() {
		wp.mu.Lock()
		if !wp.running {
			wp.mu.Unlock()
			return
		}

		wp.cancel()
		close(wp.jobChan)
		wp.mu.Unlock()

		wp.wg.Wait()

		wp.mu.Lock()
		close(wp.errChan)
		wp.running = false
		wp.mu.Unlock()
	})
}

// Errors returns a read-only channel for monitoring processing failures.
func (wp *WorkerPool) Errors() <-chan error {
	return wp.errChan
}

// IsRunning returns whether the worker pool is currently active.
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}
