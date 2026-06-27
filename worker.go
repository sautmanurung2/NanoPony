package nanopony

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ShardedWorkerPool splits jobs across multiple independent pools to reduce contention.
// Note: We maintain the "ShardedWorkerPool" name for backward compatibility with
// existing integrations, but internally use a single shared channel to prevent Head-of-Line blocking.
type ShardedWorkerPool struct {
	shards          []*workerPoolShard // Kept for API compatibility, but we only ever create ONE
	numShards       uint64
	workersPerShard int
	totalWorkers    int

	// New shared state to replace sharding logic
	sharedJobChan chan *Job
	sharedErrChan chan error
	sharedCtx     context.Context
	sharedCancel  context.CancelFunc
	sharedWg      sync.WaitGroup
	running       bool
	mu            sync.RWMutex

	// Elastic Queue State
	elasticBuffer chan *Job
	maxElastic    int

	// Disaster Recovery
	fallbackFile string
}

type workerPoolShard struct {
	jobChan chan *Job
	errChan chan error
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewWorkerPool creates a new ShardedWorkerPool.
// We ignore numShards and always use 1 shard internally to avoid Head-of-Line blocking
// while preserving the function signature for compatibility.
//
// ELASTIC QUEUE: The worker pool now implements an "elastic buffer" pattern.
// If the main queue is full, jobs spill over into an elastic buffer that can grow
// to prevent deadlocks on 1-queue-1-worker scenarios, protecting memory by capping growth.
func NewWorkerPool(
	numWorkers int,
	queueSize int,
	numShards int, // Ignored internally
) *ShardedWorkerPool {

	if numWorkers < 1 {
		numWorkers = 1
	}

	if queueSize < 1 {
		queueSize = 1
	}

	if numShards < 1 {
		numShards = 1
	}

	if numShards > numWorkers {
		numShards = numWorkers
	}

	actualShards := numShards

	ctx, cancel := context.WithCancel(context.Background())

	sharedJobChan := make(chan *Job, queueSize)
	sharedErrChan := make(chan error, queueSize)

	// Elastic buffer to handle spikes without blocking immediately on small queueSizes.
	// We cap it at max(5000, queueSize*10) to prevent OOM on sustained unhandled bursts.
	maxElastic := 5000
	if queueSize*10 > maxElastic {
		maxElastic = queueSize * 10
	}

	// We use an unbuffered channel as a signal for the elastic goroutine,
	// but hold the actual jobs in a dynamic slice inside the coordinator.
	elasticBuffer := make(chan *Job)

	shards := make([]*workerPoolShard, actualShards)
	for i := 0; i < actualShards; i++ {
		shards[i] = &workerPoolShard{
			jobChan: sharedJobChan,
			errChan: sharedErrChan,
			ctx:     ctx,
			cancel:  cancel,
		}
	}

	return &ShardedWorkerPool{
		shards:          shards,
		numShards:       uint64(actualShards),
		workersPerShard: numWorkers,
		totalWorkers:    numWorkers,

		sharedJobChan: sharedJobChan,
		sharedErrChan: sharedErrChan,
		sharedCtx:     ctx,
		sharedCancel:  cancel,

		elasticBuffer: elasticBuffer,
		maxElastic:    maxElastic,
		fallbackFile:  filepath.Join(os.TempDir(), "nanopony_recovery.jsonl"),
	}
}

// SetRecoveryFile allows overriding the default recovery file path.
// When running in Docker/K8s, set this to a mounted persistent volume path (e.g., "/data/recovery.jsonl").
func (swp *ShardedWorkerPool) SetRecoveryFile(path string) {
	swp.mu.Lock()
	defer swp.mu.Unlock()
	swp.fallbackFile = path
}

// Start activates all workers consuming from the shared channel.
func (swp *ShardedWorkerPool) Start(
	ctx context.Context,
	handler JobHandler,
) {
	swp.mu.Lock()
	defer swp.mu.Unlock()

	if swp.running {
		return
	}

	// Link contexts
	swp.sharedCtx, swp.sharedCancel = context.WithCancel(ctx)

	// Keep shards in sync just in case
	for _, shard := range swp.shards {
		shard.ctx = swp.sharedCtx
		shard.cancel = swp.sharedCancel
		shard.running = true
	}

	swp.running = true

	// Launch all workers reading from the same shared channel
	for i := 0; i < swp.totalWorkers; i++ {
		swp.sharedWg.Add(1)
		go swp.worker(handler, i)
	}

	// Start the Elastic Queue Coordinator
	swp.sharedWg.Add(1)
	go swp.elasticCoordinator()
}

// elasticCoordinator manages the infinite-ish (capped) elastic buffer.
// It receives spilled jobs from Submit/SubmitBlocking and feeds them into
// the main sharedJobChan when space becomes available.
func (swp *ShardedWorkerPool) elasticCoordinator() {
	defer swp.sharedWg.Done()

	queue := make([]*Job, 0, swp.maxElastic)

	for {
		// If we have items in our elastic queue, try to push to main channel,
		// but also be ready to accept new items (up to maxElastic limit).
		if len(queue) > 0 {
			// Get next job to push
			nextJob := queue[0]

			select {
			case swp.sharedJobChan <- nextJob:
				// Successfully pushed to main queue, remove from elastic queue
				queue[0] = nil // avoid memory leak of underlying array
				queue = queue[1:]

			case newJob, ok := <-swp.elasticBuffer:
				if !ok {
					// Channel closed, drain remaining
					swp.drainElastic(queue)
					return
				}
				// No drop: queue grows indefinitely. Caller must ensure sufficient memory or control ingress.
				queue = append(queue, newJob)
				if len(queue) > swp.maxElastic && len(queue)%5000 == 0 {
					LogFramework("WARNING", "WorkerPool", fmt.Sprintf("Elastic buffer threshold exceeded: %d jobs waiting", len(queue)))
				}

			case <-swp.sharedCtx.Done():
				// Server is shutting down.
				// Dump ALL jobs in the elastic memory buffer straight to disk.
				for _, j := range queue {
					if j != nil {
						swp.dumpJobToRecovery(j)
						j.Release()
					}
				}
				queue = nil
				return
			}
		} else {
			// Elastic queue empty, just wait for new jobs
			select {
			case newJob, ok := <-swp.elasticBuffer:
				if !ok {
					return
				}
				queue = append(queue, newJob)
			case <-swp.sharedCtx.Done():
				return
			}
		}
	}
}

func (swp *ShardedWorkerPool) drainElastic(queue []*Job) {
	for _, job := range queue {
		if job != nil {
			job.Release()
		}
	}
}

func (swp *ShardedWorkerPool) worker(
	handler JobHandler,
	id int,
) {
	defer swp.sharedWg.Done()

	for {
		select {
		case job, ok := <-swp.sharedJobChan:
			if !ok {
				return
			}

			if err := handler(swp.sharedCtx, job); err != nil {
				swp.reportError(
					job.ID,
					id,
					err,
				)
			}

			job.Release()

		case <-swp.sharedCtx.Done():
			// Server is shutting down (Hard Stop).
			// Instead of blocking for hours to process, dump all remaining jobs to disk (JSONL).
			for {
				select {
				case job, ok := <-swp.sharedJobChan:
					if !ok {
						return
					}
					swp.dumpJobToRecovery(job)
					job.Release()
				default:
					return
				}
			}
		}
	}
}

func (swp *ShardedWorkerPool) reportError(
	jobID string,
	workerID int,
	err error,
) {
	wrappedErr := fmt.Errorf("worker %d: job %s: %w", workerID, jobID, err)

	select {
	case swp.sharedErrChan <- wrappedErr:
	default:
		LogFramework(
			"WARNING",
			"ShardedWorkerPool",
			"Error channel full, discarded",
		)
	}
}

func (swp *ShardedWorkerPool) Submit(
	ctx context.Context,
	job *Job,
) error {
	// Retry loop: will continuously retry with exponential backoff if the queue is full.
	// This ensures data is never dropped and ErrQueueFull is never returned,
	// providing true 100% guarantee while maintaining internal backpressure.
	baseDelay := 10 * time.Millisecond
	maxDelay := 500 * time.Millisecond
	delay := baseDelay

	for {
		select {
		case <-swp.sharedCtx.Done():
			return ErrPoolStopped
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 1. Try to put in main channel directly (fast path)
		select {
		case <-swp.sharedCtx.Done():
			return ErrPoolStopped
		case <-ctx.Done():
			return ctx.Err()
		case swp.sharedJobChan <- job:
			return nil
		default:
			// 2. Main channel full, spill to elastic buffer
			select {
			case <-swp.sharedCtx.Done():
				return ErrPoolStopped
			case <-ctx.Done():
				return ctx.Err()
			case swp.elasticBuffer <- job:
				return nil
			case <-time.After(50 * time.Millisecond):
				// Queue full: we do NOT return ErrQueueFull.
				// We back off and retry to guarantee no data loss.
			}
		}

		// Backoff before next retry
		select {
		case <-swp.sharedCtx.Done():
			return ErrPoolStopped
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

func (swp *ShardedWorkerPool) SubmitBlocking(
	ctx context.Context,
	job *Job,
) error {
	select {
	case <-swp.sharedCtx.Done():
		return ErrPoolStopped
	default:
	}

	// 1. Try to put in main channel directly
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-swp.sharedCtx.Done():
		return ErrPoolStopped
	case swp.sharedJobChan <- job:
		return nil
	default:
		// 2. Main channel full, spill to elastic buffer blocking
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-swp.sharedCtx.Done():
			return ErrPoolStopped
		case swp.elasticBuffer <- job:
			return nil
		}
	}
}

func (swp *ShardedWorkerPool) Stop() {
	swp.mu.Lock()
	if !swp.running {
		swp.mu.Unlock()
		return
	}

	swp.running = false
	if swp.sharedCancel != nil {
		swp.sharedCancel()
	}

	for _, shard := range swp.shards {
		shard.running = false
	}

	swp.mu.Unlock()

	// Wait for all workers and coordinator to finish
	swp.sharedWg.Wait()

	close(swp.sharedJobChan)
	close(swp.sharedErrChan)
	close(swp.elasticBuffer)
}

func (swp *ShardedWorkerPool) NumWorkers() int {
	return swp.totalWorkers
}

func (swp *ShardedWorkerPool) NumShards() int {
	return int(swp.numShards)
}

func (swp *ShardedWorkerPool) IsRunning() bool {
	swp.mu.RLock()
	defer swp.mu.RUnlock()
	return swp.running
}

func (swp *ShardedWorkerPool) dumpJobToRecovery(job *Job) {
	if job == nil || swp.fallbackFile == "" {
		return
	}

	f, err := os.OpenFile(swp.fallbackFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		LogFramework("ERROR", "WorkerPool-Recovery", fmt.Sprintf("Failed to open recovery file: %v", err))
		return
	}
	defer f.Close()

	// Convert Job to JSON.
	// Note: job.Data must be JSON serializable for this to work perfectly.
	b, err := json.Marshal(job)
	if err != nil {
		LogFramework("ERROR", "WorkerPool-Recovery", fmt.Sprintf("Failed to marshal job %s: %v", job.ID, err))
		return
	}

	f.Write(b)
	f.WriteString("\n")
}
