package nanopony

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// getMemStats returns current memory allocation
func getMemStats() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// TestWorkerPoolNoMemoryLeak verifies no memory leak after multiple operations
func TestWorkerPoolNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	
	// Force GC before test
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	initialMem := getMemStats()
	t.Logf("Initial memory: %d KB", initialMem/1024)
	
	// Run multiple cycles
	for cycle := 0; cycle < 20; cycle++ {
		pool := NewWorkerPool(10, 500)
		
		pool.Start(ctx, func(ctx context.Context, job Job) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
		
		// Submit many jobs
		for i := 0; i < 100; i++ {
			pool.Submit(ctx, Job{
				ID:   "leak-test",
				Data: map[string]interface{}{"index": i, "data": make([]byte, 100)},
			})
		}
		
		time.Sleep(50 * time.Millisecond)
		pool.Stop()
	}
	
	// Force GC after test
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	
	finalMem := getMemStats()
	t.Logf("Final memory: %d KB", finalMem/1024)
	
	// Check absolute growth instead of percentage (to avoid division by small numbers)
	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth: %d KB", memGrowth/1024)
	
	// Allow max 500 KB growth (considering GC overhead)
	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

// TestPollerNoMemoryLeak verifies no memory leak in poller
func TestPollerNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(5, 500)
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	defer pool.Stop()
	
	// Force GC before test
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	initialMem := getMemStats()
	t.Logf("Initial memory: %d KB", initialMem/1024)
	
	// Run multiple poller cycles
	for cycle := 0; cycle < 20; cycle++ {
		fetchCount := 0
		fetcher := DataFetcherFunc(func() ([]interface{}, error) {
			fetchCount++
			if fetchCount > 50 {
				return []interface{}{}, nil
			}
			return []interface{}{make([]byte, 100)}, nil
		})
		
		config := DefaultPollerConfig()
		config.Interval = 10 * time.Millisecond
		
		poller := NewPoller(config, pool, fetcher)
		poller.Start()
		time.Sleep(100 * time.Millisecond)
		poller.Stop()
	}
	
	// Force GC after test
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	
	finalMem := getMemStats()
	t.Logf("Final memory: %d KB", finalMem/1024)
	
	// Check absolute growth
	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth: %d KB", memGrowth/1024)
	
	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

// TestFrameworkNoMemoryLeak verifies no memory leak in framework
func TestFrameworkNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	
	// Force GC before test
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	initialMem := getMemStats()
	t.Logf("Initial memory: %d KB", initialMem/1024)
	
	// Run multiple framework cycles
	for cycle := 0; cycle < 10; cycle++ {
		ResetConfig()
		config := NewConfig()
		
		framework := NewFramework().
			WithConfig(config).
			WithWorkerPool(5, 200)
		
		components := framework.Build()
		
		components.Start(ctx, func(ctx context.Context, job Job) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
		
		time.Sleep(100 * time.Millisecond)
		
		if err := components.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error: %v", err)
		}
	}
	
	// Force GC after test
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	
	finalMem := getMemStats()
	t.Logf("Final memory: %d KB", finalMem/1024)
	
	// Check absolute growth
	memGrowth := int64(finalMem) - int64(initialMem)
	t.Logf("Memory growth: %d KB", memGrowth/1024)
	
	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

// BenchmarkMemoryAllocation tracks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	runtime.GC()
	
	b.ReportAllocs()
	
	ctx := context.Background()
	pool := NewWorkerPool(5, 1000)
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		return nil
	})
	defer pool.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		pool.Submit(ctx, Job{
			ID:   "bench",
			Data: make([]byte, 100),
		})
	}
}
