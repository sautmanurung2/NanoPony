package nanopony

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func getMemStats() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func TestWorkerPoolNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	initialMem := getMemStats()
	for cycle := 0; cycle < 20; cycle++ {
		pool := NewWorkerPool[any](10, 500, 2)
		pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		for i := 0; i < 100; i++ {
			job := AcquireJob[any]()
			job.ID = "leak-test"
			job.Data = map[string]interface{}{"index": i, "data": make([]byte, 100)}
			pool.Submit(ctx, job)
		}

		time.Sleep(50 * time.Millisecond)
		pool.Stop()
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	finalMem := getMemStats()
	memGrowth := int64(finalMem) - int64(initialMem)

	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

func TestPollerNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool[any](5, 500, 2)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	defer pool.Stop()

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialMem := getMemStats()

	for cycle := 0; cycle < 20; cycle++ {
		fetchCount := 0
		fetcher := DataFetcherFunc[any](func() ([]any, error) {
			fetchCount++
			if fetchCount > 50 {
				return []any{}, nil
			}
			return []any{make([]byte, 100)}, nil
		})

		config := DefaultPollerConfig()
		config.Interval = 10 * time.Millisecond

		poller := NewPoller[any](config, pool, fetcher)
		poller.Start()
		time.Sleep(100 * time.Millisecond)
		poller.Stop()
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	finalMem := getMemStats()
	memGrowth := int64(finalMem) - int64(initialMem)

	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

func TestFrameworkNoMemoryLeak(t *testing.T) {
	ctx := context.Background()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialMem := getMemStats()

	for cycle := 0; cycle < 10; cycle++ {
		ResetConfig()
		config := NewConfig()
		framework := NewFramework[any]().
			WithConfig(config).
			WithWorkerPool(5, 200, 2)

		components := framework.Build()
		components.Start(ctx, func(ctx context.Context, job *Job[any]) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		time.Sleep(100 * time.Millisecond)
		_ = components.Shutdown(ctx)
	}

	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	finalMem := getMemStats()
	memGrowth := int64(finalMem) - int64(initialMem)

	if memGrowth > 500*1024 {
		t.Errorf("Potential memory leak detected: %d KB growth", memGrowth/1024)
	}
}

func BenchmarkMemoryAllocation(b *testing.B) {
	runtime.GC()
	b.ReportAllocs()
	ctx := context.Background()
	pool := NewWorkerPool[any](5, 1000, 2)
	pool.Start(ctx, func(ctx context.Context, job *Job[any]) error {
		return nil
	})
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := AcquireJob[any]()
		job.ID = "bench"
		job.Data = make([]byte, 100)
		pool.Submit(ctx, job)
	}
}
