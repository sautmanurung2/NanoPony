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
		shards[i] = &workerPoolShard{
			jobChan: make(chan *Job, shardQueue),
			errChan: make(chan error, shardQueue),
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

func (swp *ShardedWorkerPool) reportError(
	shard *workerPoolShard,
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

func (swp *ShardedWorkerPool) getShard() *workerPoolShard {
	idx := atomic.AddUint64(&swp.nextShard, 1) % swp.numShards
	return swp.shards[idx]
}

func (swp *ShardedWorkerPool) Submit(
	ctx context.Context,
	job *Job,
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

func (swp *ShardedWorkerPool) SubmitBlocking(
	ctx context.Context,
	job *Job,
) error {
	shard := swp.getShard()

	select {
	case <-ctx.Done():
		return ctx.Err()

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

		close(shard.jobChan)

		shard.mu.Unlock()

		shard.wg.Wait()

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
