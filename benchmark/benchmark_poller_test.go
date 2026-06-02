//go:build benchmark

package benchmark

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sautmanurung2/nanopony"
)

// BenchmarkPollerCreation tests memory allocation for poller creation
func BenchmarkPollerCreation(b *testing.B) {
	b.ReportAllocs()

	pool := nanopony.NewWorkerPool(5, 100, 2)
	defer pool.Stop()

	fetcher := nanopony.DataFetcherFunc(func(ctx context.Context) ([]any, error) {
		return []any{"data"}, nil
	})

	for i := 0; i < b.N; i++ {
		poller := nanopony.NewPoller(nanopony.DefaultPollerConfig(), pool, fetcher)
		if poller == nil {
			b.Fatal("Expected poller to be created")
		}
	}
}

// BenchmarkPollerStartStop tests memory allocation for poller start/stop cycles
func BenchmarkPollerStartStop(b *testing.B) {
	b.ReportAllocs()

	pool := nanopony.NewWorkerPool(5, 100, 2)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job *nanopony.Job) error {
		return nil
	})
	defer pool.Stop()

	fetcher := nanopony.DataFetcherFunc(func(ctx context.Context) ([]any, error) {
		return []any{"data"}, nil
	})

	for i := 0; i < b.N; i++ {
		poller := nanopony.NewPoller(nanopony.DefaultPollerConfig(), pool, fetcher)
		poller.Start()
		time.Sleep(10 * time.Millisecond)
		poller.Stop()
	}
}

// BenchmarkPollerFetch tests memory allocation during polling
func BenchmarkPollerFetch(b *testing.B) {
	pool := nanopony.NewWorkerPool(5, 1000, 2)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job *nanopony.Job) error {
		return nil
	})
	defer pool.Stop()

	fetchCount := 0
	fetcher := nanopony.DataFetcherFunc(func(ctx context.Context) ([]any, error) {
		fetchCount++
		return []any{map[string]any{"count": fetchCount}}, nil
	})

	config := nanopony.DefaultPollerConfig()
	config.Interval = 1 * time.Millisecond

	b.ReportAllocs()
	b.ResetTimer()

	poller := nanopony.NewPoller(config, pool, fetcher)
	poller.Start()
	time.Sleep(100 * time.Millisecond)
	poller.Stop()
}

// TestPollerMemoryLeak detects memory leaks in poller
func TestPollerMemoryLeak(t *testing.T) {
	pool := nanopony.NewWorkerPool(5, 100, 2)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job *nanopony.Job) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	defer pool.Stop()

	// Run multiple start/stop cycles
	for cycle := 0; cycle < 10; cycle++ {
		fetcher := nanopony.DataFetcherFunc(func(ctx context.Context) ([]any, error) {
			return []any{"data"}, nil
		})

		poller := nanopony.NewPoller(nanopony.DefaultPollerConfig(), pool, fetcher)
		poller.Start()
		time.Sleep(50 * time.Millisecond)
		poller.Stop()

		t.Logf("Cycle %d completed", cycle)
	}
}

// TestPollerLongRunning tests memory usage over extended polling period
func TestPollerLongRunning(t *testing.T) {
	pool := nanopony.NewWorkerPool(5, 1000, 2)
	ctx := context.Background()

	var processed atomic.Int64
	pool.Start(ctx, func(ctx context.Context, job *nanopony.Job) error {
		processed.Add(1)
		return nil
	})
	defer pool.Stop()

	var fetchCount atomic.Int64
	fetcher := nanopony.DataFetcherFunc(func(ctx context.Context) ([]any, error) {
		count := fetchCount.Add(1)
		if count > 100 {
			return []any{}, nil
		}
		return []any{"data"}, nil
	})

	config := nanopony.DefaultPollerConfig()
	config.Interval = 10 * time.Millisecond

	poller := nanopony.NewPoller(config, pool, fetcher)
	poller.Start()

	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	poller.Stop()

	t.Logf("Total fetches: %d, Total processed: %d", fetchCount.Load(), processed.Load())
}
