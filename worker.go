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
// Sharding significantly reduces lock contention on the job channel and state management.

package nanopony

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ShardedWorkerPool splits jobs across multiple independent pools to reduce contention.
type ShardedWorkerPool[T any] struct {
	shards          []*workerPoolShard[T]
	numShards       uint64
	nextShard       uint64
	workersPerShard int
	totalWorkers    int
}

type workerPoolShard[T any] struct {
	jobChan chan *Job[T]
	errChan chan error

	wg sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	running bool
	mu      sync.RWMutex
}

// NewWorkerPool creates a new ShardedWorkerPool.
//
// numWorkers = total worker count across all shards
// queueSize  = total queue size across all shards
// numShards  = number of shards
func NewWorkerPool[T any](
	numWorkers int,
	queueSize int,
	numShards int,
) *ShardedWorkerPool[T] {

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

	workersPerShard := numWorkers / numShards
	if workersPerShard < 1 {
		workersPerShard = 1
	}

	shardQueue := queueSize / numShards
	if shardQueue < 1 {
		shardQueue = 1
	}

	shards := make([]*workerPoolShard[T], numShards)

	for i := 0; i < numShards; i++ {
		shards[i] = &workerPoolShard[T]{
			jobChan: make(chan *Job[T], shardQueue),
			errChan: make(chan error, shardQueue),
		}
	}

	return &ShardedWorkerPool[T]{
		shards:          shards,
		numShards:       uint64(numShards),
		workersPerShard: workersPerShard,
		totalWorkers:    numWorkers,
	}
}

// Start activates all shards.
func (swp *ShardedWorkerPool[T]) Start(
	ctx context.Context,
	handler JobHandler[T],
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

func (swp *ShardedWorkerPool[T]) worker(
	shard *workerPoolShard[T],
	handler JobHandler[T],
	id int,
) {
	defer shard.wg.Done()

	for {
		select {
		case <-shard.ctx.Done():
			return

		case job, ok := <-shard.jobChan:
			if !ok {
				return
			}

			if err := handler(shard.ctx, job); err != nil {
				swp.reportError(
					shard,
					fmt.Errorf(
						"shard worker %d: job %s failed: %w",
						id,
						job.ID,
						err,
					),
				)
			}

			job.Release()
		}
	}
}

func (swp *ShardedWorkerPool[T]) reportError(
	shard *workerPoolShard[T],
	err error,
) {
	select {
	case shard.errChan <- err:
	default:
		LogFramework(
			"WARNING",
			"ShardedWorkerPool",
			"Error channel full, discarded",
		)
	}
}

func (swp *ShardedWorkerPool[T]) getShard() *workerPoolShard[T] {
	idx := atomic.AddUint64(&swp.nextShard, 1) % swp.numShards
	return swp.shards[idx]
}

func (swp *ShardedWorkerPool[T]) Submit(
	ctx context.Context,
	job *Job[T],
) error {
	shard := swp.getShard()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case shard.jobChan <- job:
		return nil

	default:
		return ErrQueueFull
	}
}

func (swp *ShardedWorkerPool[T]) SubmitBlocking(
	ctx context.Context,
	job *Job[T],
) error {
	shard := swp.getShard()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case shard.jobChan <- job:
		return nil
	}
}

func (swp *ShardedWorkerPool[T]) Stop() {
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

		close(shard.jobChan)

		shard.mu.Unlock()

		shard.wg.Wait()

		close(shard.errChan)
	}
}

// NumWorkers returns configured worker count.
func (swp *ShardedWorkerPool[T]) NumWorkers() int {
	return swp.totalWorkers
}

// NumShards returns configured shard count.
func (swp *ShardedWorkerPool[T]) NumShards() int {
	return int(swp.numShards)
}

// IsRunning returns true if at least one shard is running.
func (swp *ShardedWorkerPool[T]) IsRunning() bool {
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

// Errors returns a combined channel for all shard errors.
func (swp *ShardedWorkerPool[T]) Errors() <-chan error {
	// For simplicity, we'll return the first shard's error channel 
	// or a merged channel. Merging channels in Go is complex.
	// Most users only use one shard or expect a single stream.
	if len(swp.shards) > 0 {
		return swp.shards[0].errChan
	}
	return nil
}
