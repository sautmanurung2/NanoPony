> **Language:** [Bahasa Indonesia](WORKER_POOL_EXPLAINED.md) | [English](WORKER_POOL_EXPLAINED_EN.md)

# Sharded Worker Pool Architecture

## Summary

This document explains the *sharded worker pool* architecture in the NanoPony framework. NanoPony uses a *sharding* system to reduce *lock contention* and recycles `Job` objects using `sync.Pool` for memory efficiency.

---

## Architecture: Sharded Pool

NanoPony now uses a Shared Queue architecture with an Elastic Buffer to prevent Head-of-Line blocking while retaining the `ShardedWorkerPool` name for compatibility.

```text
                               ┌──→ Shard 0 (Pool 0) ──→ Worker...
Submit ──→ Shared Elastic Queue ──→ Shared Channel ──→ Workers...
```

When `Submit()` is called, the job is pushed into the shared channel. If the main queue is full, the job is moved to the Elastic Buffer memory with an exponential backoff mechanism, ensuring 100% reliability without data loss.

---

## 1. Sharding Implementation

```go
// Sharded Worker Pool
type ShardedWorkerPool struct {
    shards          []*workerPoolShard
    // ...
}

// WorkerPoolShard
type workerPoolShard struct {
    jobChan chan *Job
    errChan chan error
    // ...
}
```

*   **Elastic Queue**: With the Elastic Buffer mechanism, job spikes are handled without instant deadlocks, providing system resilience when the main queue is full.

---

## 2. Job Recycling (sync.Pool)

NanoPony v0.0.59 introduces `sync.Pool` for the `Job` struct.

### Why is this important?
In high-throughput scenarios (e.g., 10,000 jobs/second), creating a new `*Job` object for each item causes excessive heap allocation, leading to GC (Garbage Collection) overhead.

### How it works:
1.  **Acquisition**: `job := nanopony.AcquireJob()` takes an existing object from the pool.
2.  **Reset**: Before being returned, `job.Release()` clears data to prevent data leakage.
3.  **Reuse**: The worker automatically calls `job.Release()` after the `JobHandler` finishes processing it.

---

## 3. Worker Execution Flow

```go
func (swp *ShardedWorkerPool) worker(shard *workerPoolShard, handler JobHandler, wg *sync.WaitGroup) {
    defer wg.Done()
    for {
        select {
        case <-swp.ctx.Done():
            return
        case job, ok := <-shard.jobChan:
            if !ok {
                return
            }
            // 1. Process Job
            err := handler(swp.ctx, job)
            
            // 2. Report Error (non-blocking)
            if err != nil {
                select {
                case shard.errChan <- err:
                default:
                }
            }
            
            // 3. Return Job to Pool
            job.Release()
        }
    }
}
```

| Aspect | Behavior |
|-------|----------|
| **Concurrency** | *Single shared channel* avoids *Head-of-Line blocking* issues. |
| **Blocking** | *Worker* is synchronous per *job* (waits for *handler* to finish). |
| **Locking** | Optimal thanks to the use of a shared queue with a buffer. |
| **Submit** | Uses a backoff mechanism and *Elastic Buffer* to guarantee the *queue* drops no jobs. |

---

## 4. Conclusion

This system combines the reliability of a *worker pool* mechanism with high scalability through a shared single channel and an *Elastic Buffer*. This design preserves the *backpressure* principle to maintain application stability under high workloads.
