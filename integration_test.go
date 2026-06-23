package nanopony

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegrationPollerWorkerPool tests the integration between Poller, WorkerPool, and Job execution.
func TestIntegrationPollerWorkerPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var processedCount int32
	expectedCount := int32(50)

	// Create WorkerPool
	pool := NewWorkerPool(5, 100, 2)
	pool.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processedCount, 1)
		time.Sleep(1 * time.Millisecond) // Simulate work
		return nil
	})
	defer pool.Stop()

	// Create Poller with a mock fetcher
	var fetchCount int32
	fetcher := DataFetcherFunc(func(ctx context.Context) ([]interface{}, error) {
		count := atomic.AddInt32(&fetchCount, 1)
		if count > 5 {
			return nil, nil // Stop fetching
		}

		// Return 10 items per fetch
		items := make([]interface{}, 10)
		for i := 0; i < 10; i++ {
			items[i] = map[string]interface{}{"id": count*10 + int32(i)}
		}
		return items, nil
	})

	config := DefaultPollerConfig()
	config.Interval = 10 * time.Millisecond
	config.BatchSize = 10
	config.JobSlotSize = 1

	poller := NewPoller(config, pool, fetcher)
	poller.Start()
	defer poller.Stop()

	// Wait for jobs to be processed
	time.Sleep(500 * time.Millisecond)

	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != expectedCount {
		t.Errorf("Integration test failed: expected %d jobs processed, got %d", expectedCount, finalCount)
	}
}

// TestIntegrationFrameworkLifecycle tests the full lifecycle of the Framework using components.
func TestIntegrationFrameworkLifecycle(t *testing.T) {
	// Setup config
	ResetConfig()
	config := BuildConfig(func(c *Config) {
		c.App.Env = "testing"
		c.App.LogOutputMode = "console"
	})

	// Setup framework with basic components (excluding Database/Kafka which require external services)
	fw := NewFramework().
		WithConfig(config).
		WithWorkerPool(2, 50, 1)

	components, err := fw.BuildSafe()
	if err != nil {
		t.Fatalf("Failed to build framework: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var started int32
	err = components.Start(ctx, func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&started, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to start components: %v", err)
	}

	// Submit a job to ensure worker pool is active
	job := AcquireJob()
	job.ID = "integration-job-1"
	job.Data = "test"
	_ = components.WorkerPool.Submit(ctx, job)

	time.Sleep(100 * time.Millisecond)

	// Shutdown framework
	err = components.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown framework: %v", err)
	}

	if atomic.LoadInt32(&started) != 1 {
		t.Errorf("Expected 1 job to be processed, got %d", started)
	}
}
