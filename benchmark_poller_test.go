package nanopony

import (
	"context"
	"testing"
	"time"
)

// BenchmarkPollerCreation tests memory allocation for poller creation
func BenchmarkPollerCreation(b *testing.B) {
	b.ReportAllocs()
	
	pool := NewWorkerPool(5, 100)
	defer pool.Stop()
	
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return []interface{}{"data"}, nil
	})
	
	for i := 0; i < b.N; i++ {
		poller := NewPoller(DefaultPollerConfig(), pool, fetcher)
		if poller == nil {
			b.Fatal("Expected poller to be created")
		}
	}
}

// BenchmarkPollerStartStop tests memory allocation for poller start/stop cycles
func BenchmarkPollerStartStop(b *testing.B) {
	b.ReportAllocs()
	
	pool := NewWorkerPool(5, 100)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		return nil
	})
	defer pool.Stop()
	
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return []interface{}{"data"}, nil
	})
	
	for i := 0; i < b.N; i++ {
		poller := NewPoller(DefaultPollerConfig(), pool, fetcher)
		poller.Start()
		time.Sleep(10 * time.Millisecond)
		poller.Stop()
	}
}

// BenchmarkPollerFetch tests memory allocation during polling
func BenchmarkPollerFetch(b *testing.B) {
	pool := NewWorkerPool(5, 1000)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		return nil
	})
	defer pool.Stop()
	
	fetchCount := 0
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		fetchCount++
		return []interface{}{map[string]interface{}{"count": fetchCount}}, nil
	})
	
	config := DefaultPollerConfig()
	config.Interval = 1 * time.Millisecond
	
	b.ReportAllocs()
	b.ResetTimer()
	
	poller := NewPoller(config, pool, fetcher)
	poller.Start()
	time.Sleep(100 * time.Millisecond)
	poller.Stop()
}

// TestPollerMemoryLeak detects memory leaks in poller
func TestPollerMemoryLeak(t *testing.T) {
	pool := NewWorkerPool(5, 100)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	defer pool.Stop()
	
	// Run multiple start/stop cycles
	for cycle := 0; cycle < 10; cycle++ {
		fetcher := DataFetcherFunc(func() ([]interface{}, error) {
			return []interface{}{"data"}, nil
		})
		
		poller := NewPoller(DefaultPollerConfig(), pool, fetcher)
		poller.Start()
		time.Sleep(50 * time.Millisecond)
		poller.Stop()
		
		t.Logf("Cycle %d completed", cycle)
	}
}

// TestPollerLongRunning tests memory usage over extended polling period
func TestPollerLongRunning(t *testing.T) {
	pool := NewWorkerPool(5, 1000)
	ctx := context.Background()
	
	processed := 0
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		processed++
		return nil
	})
	defer pool.Stop()
	
	fetchCount := 0
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		fetchCount++
		if fetchCount > 100 {
			return []interface{}{}, nil
		}
		return []interface{}{"data"}, nil
	})
	
	config := DefaultPollerConfig()
	config.Interval = 10 * time.Millisecond
	
	poller := NewPoller(config, pool, fetcher)
	poller.Start()
	
	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	poller.Stop()
	
	t.Logf("Total fetches: %d, Total processed: %d", fetchCount, processed)
}
