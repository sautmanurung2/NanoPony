package nanopony

import (
	"context"
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
