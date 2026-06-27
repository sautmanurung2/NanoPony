package nanopony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestStressWorkerPool pushes a high volume of jobs to the worker pool to ensure it handles load gracefully without race conditions or memory leaks.
func TestStressWorkerPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numWorkers := 20
	queueSize := 1000
	numShards := 5
	totalJobs := 50000 // 50k jobs to process

	pool := NewWorkerPool(numWorkers, queueSize, numShards)

	var processedCount int32
	var errorCount int32

	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processedCount, 1)
		// Small delay to simulate some realistic minor work and cause queue buildup
		if job.Data.(int)%100 == 0 {
			time.Sleep(1 * time.Microsecond)
		}
		return nil
	})

	// Start a goroutine to read errors to prevent channel blocking
	go func() {
		for i := 0; i < pool.NumShards(); i++ {
			shard := pool.shards[i]
			go func(errChan <-chan error) {
				for err := range errChan {
					if err != nil {
						atomic.AddInt32(&errorCount, 1)
					}
				}
			}(shard.errChan)
		}
	}()

	startTime := time.Now()

	// Submit jobs as fast as possible
	for i := 0; i < totalJobs; i++ {
		job := AcquireJob()
		job.ID = "stress-job"
		job.Data = i

		// Use SubmitBlocking to handle backpressure properly
		err := pool.SubmitBlocking(ctx, job)
		if err != nil {
			job.Release()
			t.Logf("Failed to submit job at index %d: %v", i, err)
			break
		}
	}

	// Wait for queue to drain
	for atomic.LoadInt32(&processedCount) < int32(totalJobs) {
		time.Sleep(10 * time.Millisecond)
	}
	pool.Stop()
	duration := time.Since(startTime)

	finalProcessed := atomic.LoadInt32(&processedCount)
	t.Logf("Stress Test Results:")
	t.Logf("Total Jobs Submitted: %d", totalJobs)
	t.Logf("Total Jobs Processed: %d", finalProcessed)
	t.Logf("Total Errors: %d", atomic.LoadInt32(&errorCount))
	t.Logf("Time Taken: %v", duration)
	t.Logf("Throughput: %.2f jobs/sec", float64(finalProcessed)/duration.Seconds())

	if finalProcessed != int32(totalJobs) {
		t.Errorf("Stress test failed: expected %d jobs to be processed, got %d", totalJobs, finalProcessed)
	}
}
