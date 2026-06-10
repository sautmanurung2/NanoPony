// worker.go — Concurrent job processing using a sharded worker pool.
//
// Architecture (fan-out pattern with sharding):
//
//	┌─── shard 0 (pool 0) ──→ workerChan ──→ worker...
//
// Submit ─┼─── shard 1 (pool 1) ──→ workerChan ──→ worker...
//
//	└─── shard N (pool N) ──→ workerChan ──→ worker...
//
// Each shard operates independently with its own job channel and worker goroutines.
// Each shard has its own job channel and worker goroutines. Jobs are distributed
// across shards in a round-robin manner.
// Sharding significantly reduces lock contention on the job channel and state management.
package nanopony

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ShardedWorkerPool splits jobs across multiple independent pools to reduce contention.
type ShardedWorkerPool struct {
	shards          []*workerPoolShard
	numShards       uint64
	nextShard       uint64
	workersPerShard int
	totalWorkers    int
}

type workerPoolShard struct {
	jobChan chan *Job
	errChan chan error
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

// NewWorkerPool creates a new ShardedWorkerPool.
//
// numWorkers = total worker count across all shards
// queueSize  = total queue size across all shards
// numShards  = number of shards
func NewWorkerPool(
	numWorkers int,
	queueSize int,
	numShards int,
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

	// Tidak masuk akal shard lebih banyak daripada worker.
	if numShards > numWorkers {
		numShards = numWorkers
	}

	workersPerShard := numWorkers / numShards
	if workersPerShard < 1 {
		workersPerShard = 1
	}

	shardQueue := queueSize / numShards
	if shardQueue < 1 {
		shardQueue = 1
	}

	shards := make([]*workerPoolShard, numShards)

	for i := 0; i < numShards; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		shards[i] = &workerPoolShard{
			jobChan: make(chan *Job, shardQueue),
			errChan: make(chan error, shardQueue),
			ctx:     ctx,
			cancel:  cancel,
		}
	}

	return &ShardedWorkerPool{
		shards:          shards,
		numShards:       uint64(numShards),
		workersPerShard: workersPerShard,
		totalWorkers:    numWorkers,
	}
}

// Start activates all shards.
func (swp *ShardedWorkerPool) Start(
	ctx context.Context,
	handler JobHandler,
) {
	for _, shard := range swp.shards {
		shard.mu.Lock()

		if shard.running {
			shard.mu.Unlock()
			continue
		}

		shard.ctx, shard.cancel = context.WithCancel(ctx)
		shard.running = true

		for i := 0; i < swp.workersPerShard; i++ {
			shard.wg.Add(1)
			go swp.worker(shard, handler, i)
		}

		shard.mu.Unlock()
	}
}

func (swp *ShardedWorkerPool) worker(
	shard *workerPoolShard,
	handler JobHandler,
	id int,
) {
	defer shard.wg.Done()

	for {
		select {
		case job, ok := <-shard.jobChan:
			if !ok {
				return
			}

			if err := handler(shard.ctx, job); err != nil {
				swp.reportError(
					shard,
					job.ID,
					id,
					err,
				)
			}

			job.Release()

		case <-shard.ctx.Done():
			// Context canceled, drain remaining jobs in the channel to prevent sync.Pool leaks.
			for {
				select {
				case job, ok := <-shard.jobChan:
					if !ok {
						return
					}
					// Still call handler so it can see the canceled context and potentially
					// perform fast cleanup/skip, then release the job.
					_ = handler(shard.ctx, job)
					job.Release()
				default:
					return
				}
			}
		}
	}
}

func (swp *ShardedWorkerPool) reportError(
	shard *workerPoolShard,
	jobID string,
	workerID int,
	err error,
) {
	// Wrapped error construction is moved here to keep the worker loop clean.
	// We use a simple wrap to minimize overhead.
	wrappedErr := fmt.Errorf("shard worker %d: job %s: %w", workerID, jobID, err)

	select {
	case shard.errChan <- wrappedErr:
	default:
		LogFramework(
			"WARNING",
			"ShardedWorkerPool",
			"Error channel full, discarded",
		)
	}
}

func (swp *ShardedWorkerPool) getShard() *workerPoolShard {
	idx := atomic.AddUint64(&swp.nextShard, 1) % swp.numShards
	return swp.shards[idx]
}

func (swp *ShardedWorkerPool) Submit(
	ctx context.Context,
	job *Job,
) error {
	shard := swp.getShard()

	// Check if pool is stopped before attempting to send
	select {
	case <-shard.ctx.Done():
		return ErrPoolStopped
	default:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-shard.ctx.Done():
		return ErrPoolStopped

	case shard.jobChan <- job:
		return nil

	default:
		return ErrQueueFull
	}
}

func (swp *ShardedWorkerPool) SubmitBlocking(
	ctx context.Context,
	job *Job,
) error {
	shard := swp.getShard()

	// Check if pool is stopped before attempting to send
	select {
	case <-shard.ctx.Done():
		return ErrPoolStopped
	default:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-shard.ctx.Done():
		return ErrPoolStopped

	case shard.jobChan <- job:
		return nil
	}
}


func (swp *ShardedWorkerPool) Stop() {
	for _, shard := range swp.shards {
		shard.mu.Lock()

		if !shard.running {
			shard.mu.Unlock()
			continue
		}

		shard.running = false

		if shard.cancel != nil {
			shard.cancel()
		}

		shard.mu.Unlock()

		// Wait for workers to finish current jobs and drain the queue.
		shard.wg.Wait()

		// Now it's safe to close channels as no more jobs will be submitted
		// and all workers have exited.
		close(shard.jobChan)
		close(shard.errChan)
	}
}

// NumWorkers returns configured worker count.
func (swp *ShardedWorkerPool) NumWorkers() int {
	return swp.totalWorkers
}

// NumShards returns configured shard count.
func (swp *ShardedWorkerPool) NumShards() int {
	return int(swp.numShards)
}

// IsRunning returns true if at least one shard is running.
func (swp *ShardedWorkerPool) IsRunning() bool {
	for _, shard := range swp.shards {
		shard.mu.RLock()
		running := shard.running
		shard.mu.RUnlock()

		if running {
			return true
		}
	}

	return false
}
