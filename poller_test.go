package nanopony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollerStartStop(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		return nil
	})
	defer pool.Stop()

	fetchCount := 0
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		fetchCount++
		if fetchCount >= 3 {
			return []interface{}{}, nil
		}
		return []interface{}{"data1"}, nil
	})

	config := DefaultPollerConfig()
	config.Interval = 50 * time.Millisecond
	config.JobSlotSize = 1

	poller := NewPoller(config, pool, fetcher)
	poller.Start()

	if !poller.IsRunning() {
		t.Error("Expected poller to be running")
	}

	time.Sleep(200 * time.Millisecond)
	poller.Stop()

	if poller.IsRunning() {
		t.Error("Expected poller to be stopped")
	}
}

func TestDataFetcherFunc(t *testing.T) {
	expected := []interface{}{"test1", "test2"}
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return expected, nil
	})

	result, err := fetcher.Fetch()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
	}
}

func TestPollerConfig(t *testing.T) {
	config := DefaultPollerConfig()

	if config.Interval != 1*time.Second {
		t.Errorf("Expected interval 1s, got %v", config.Interval)
	}
	if config.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.MaxRetries)
	}
	if config.BatchSize != 100 {
		t.Errorf("Expected batch size 100, got %d", config.BatchSize)
	}
}

func TestPollerBatchSizeLimit(t *testing.T) {
	pool := NewWorkerPool(1, 5)
	ctx := context.Background()

	var processed atomic.Int64
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(10 * time.Millisecond)
		processed.Add(1)
		return nil
	})
	defer pool.Stop()

	// Create fetcher that returns more data than batch size
	var fetchCount atomic.Int64
	fetcher := DataFetcherFunc(func() ([]any, error) {
		count := fetchCount.Add(1)
		if count >= 3 {
			return []any{}, nil
		}
		// Return 20 items, but batch size will limit to 5
		data := make([]any, 20)
		for i := range data {
			data[i] = "item"
		}
		return data, nil
	})

	config := DefaultPollerConfig()
	config.Interval = 50 * time.Millisecond
	config.JobSlotSize = 1
	config.BatchSize = 5 // Limit batch size

	poller := NewPoller(config, pool, fetcher)
	poller.Start()

	// Wait for polling to complete
	time.Sleep(200 * time.Millisecond)
	poller.Stop()

	// Each poll should only process 5 items (not 20)
	// With 2 successful polls, should have ~10 items processed
	if processed.Load() > 15 {
		t.Errorf("Expected ~10 jobs processed (limited by BatchSize=5), got %d", processed.Load())
	}
}

func TestPollerBlockingSubmitPreventsJobLoss(t *testing.T) {
	pool := NewWorkerPool(1, 3)
	ctx := context.Background()

	var processed atomic.Int64
	pool.Start(ctx, func(ctx context.Context, job Job) error {
		time.Sleep(10 * time.Millisecond)
		processed.Add(1)
		return nil
	})
	defer pool.Stop()

	// Create fetcher that returns 10 items
	var fetchCount atomic.Int64
	fetcher := DataFetcherFunc(func() ([]any, error) {
		count := fetchCount.Add(1)
		if count >= 2 {
			return []any{}, nil
		}
		data := make([]any, 10)
		for i := range data {
			data[i] = "item"
		}
		return data, nil
	})

	config := DefaultPollerConfig()
	config.Interval = 50 * time.Millisecond
	config.JobSlotSize = 1
	config.BatchSize = 10

	poller := NewPoller(config, pool, fetcher)
	poller.Start()

	// Wait for all jobs to be processed
	time.Sleep(300 * time.Millisecond)
	poller.Stop()

	// ALL jobs should be processed (no job loss with SubmitBlocking)
	expectedJobs := int64(10) // 1 poll * 10 items
	if processed.Load() != expectedJobs {
		t.Errorf("Expected %d jobs processed (no job loss), got %d", expectedJobs, processed.Load())
	}
}
