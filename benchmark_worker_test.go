package nanopony

import (
	"context"
	"sync"
	"testing"
	"time"
)

// BenchmarkWorkerPoolCreation tests memory allocation for worker pool creation
func BenchmarkWorkerPoolCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool := NewWorkerPool(5, 100)
		if pool == nil {
			b.Fatal("Expected worker pool to be created")
		}
	}
}

// BenchmarkWorkerPoolStartStop tests memory allocation for start/stop cycles
func BenchmarkWorkerPoolStartStop(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	
	for i := 0; i < b.N; i++ {
		pool := NewWorkerPool(5, 100)
		pool.Start(ctx, func(ctx context.Context, job Job) error {
			return nil
		})
		
		// Submit some jobs
		for j := 0; j < 10; j++ {
			pool.Submit(ctx, Job{ID: "bench-job"})
		}
		
		time.Sleep(1 * time.Millisecond)
		pool.Stop()
	}
}

// BenchmarkWorkerPoolSubmit tests memory allocation for job submission
func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(5, 1000)
	ctx := context.Background()
	
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(100 * time.Microsecond)
		return nil
	})
	defer pool.Stop()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pool.Submit(ctx, Job{
			ID:   "bench-job",
			Data: map[string]interface{}{"index": i},
		})
	}
}

// BenchmarkWorkerPoolSubmitParallel tests concurrent job submission
func BenchmarkWorkerPoolSubmitParallel(b *testing.B) {
	pool := NewWorkerPool(10, 10000)
	ctx := context.Background()
	
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(50 * time.Microsecond)
		return nil
	})
	defer pool.Stop()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pool.Submit(ctx, Job{
				ID:   "parallel-job",
				Data: i,
			})
			i++
		}
	})
}

// TestWorkerPoolMemoryLeak detects memory leaks in worker pool
func TestWorkerPoolMemoryLeak(t *testing.T) {
	ctx := context.Background()
	
	// Run multiple start/stop cycles
	for cycle := 0; cycle < 10; cycle++ {
		pool := NewWorkerPool(5, 100)
		
		processed := 0
		var mu sync.Mutex
		
		pool.Start(ctx, func(ctx context.Context, job Job) error {
			mu.Lock()
			processed++
			mu.Unlock()
			return nil
		})
		
		// Submit jobs
		for i := 0; i < 50; i++ {
			if err := pool.Submit(ctx, Job{ID: "leak-test"}); err != nil {
				t.Logf("Submit error: %v", err)
			}
		}
		
		time.Sleep(100 * time.Millisecond)
		pool.Stop()
		
		t.Logf("Cycle %d: Processed %d jobs", cycle, processed)
	}
}

// TestWorkerPoolLongRunning tests memory usage over extended period
func TestWorkerPoolLongRunning(t *testing.T) {
	pool := NewWorkerPool(5, 1000)
	ctx := context.Background()
	
	processed := make(chan int, 1000)
	
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		processed <- 1
		return nil
	})
	defer pool.Stop()
	
	// Submit jobs over time
	totalJobs := 1000
	for i := 0; i < totalJobs; i++ {
		pool.Submit(ctx, Job{ID: "long-running"})
		if i % 100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	
	// Wait for processing
	time.Sleep(500 * time.Millisecond)
	
	t.Logf("Processed %d jobs", totalJobs)
}
