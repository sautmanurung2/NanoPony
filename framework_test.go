package nanopony

import (
	"context"
	"testing"
)

func TestNewFramework(t *testing.T) {
	framework := NewFramework()
	if framework == nil {
		t.Fatal("Expected framework to be created")
	}
}

func TestFrameworkBuilder(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	framework := NewFramework().
		WithConfig(config)

	if framework.config != config {
		t.Error("Expected config to be set")
	}
}

func TestFrameworkWithDatabaseFromInstance(t *testing.T) {
	// Create a mock database connection (nil for testing)
	framework := NewFramework().
		WithDatabaseFromInstance(nil)

	if framework.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestFrameworkWithKafkaWriterFromInstance(t *testing.T) {
	framework := NewFramework().
		WithKafkaWriterFromInstance(nil)

	if framework.kafkaWriter != nil {
		t.Error("Expected kafkaWriter to be nil")
	}
}

func TestFrameworkWithProducerFromInstance(t *testing.T) {
	framework := NewFramework().
		WithProducerFromInstance(nil)

	if framework.producer != nil {
		t.Error("Expected producer to be nil")
	}
}

func TestFrameworkWithWorkerPoolFromInstance(t *testing.T) {
	pool := NewWorkerPool(5, 100)
	framework := NewFramework().
		WithWorkerPoolFromInstance(pool)

	if framework.workerPool != pool {
		t.Error("Expected workerPool to be set")
	}
}

func TestFrameworkWithPollerFromInstance(t *testing.T) {
	pool := NewWorkerPool(5, 100)
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return []interface{}{}, nil
	})
	poller := NewPoller(DefaultPollerConfig(), pool, fetcher)

	framework := NewFramework().
		WithPollerFromInstance(poller)

	if framework.poller != poller {
		t.Error("Expected poller to be set")
	}
}


func TestFrameworkAddCleanup(t *testing.T) {
	cleanupCalled := false
	framework := NewFramework().
		AddCleanup(func() error {
			cleanupCalled = true
			return nil
		})

	if len(framework.cleanupFuncs) != 1 {
		t.Errorf("Expected 1 cleanup function, got %d", len(framework.cleanupFuncs))
	}

	// Call cleanup
	framework.cleanupFuncs[0]()
	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}
}

func TestFrameworkBuild(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	framework := NewFramework().
		WithConfig(config).
		WithWorkerPool(5, 100)

	components := framework.Build()

	if components.Config != config {
		t.Error("Expected config to be set in components")
	}
	if components.WorkerPool == nil {
		t.Error("Expected worker pool to be set in components")
	}
}

func TestFrameworkBuildPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for double build")
		}
	}()

	framework := NewFramework()
	framework.Build()
	framework.Build() // Should panic
}

func TestFrameworkComponentsStartStop(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return []interface{}{}, nil
	})
	poller := NewPoller(DefaultPollerConfig(), pool, fetcher)

	components := &FrameworkComponents{
		WorkerPool: pool,
		Poller:     poller,
	}

	ctx := context.Background()
	components.Start(ctx, func(ctx context.Context, job Job) error {
		return nil
	})

	if !pool.IsRunning() {
		t.Error("Expected worker pool to be running")
	}
	if !poller.IsRunning() {
		t.Error("Expected poller to be running")
	}

	err := components.Shutdown(ctx)
	if err != nil {
		t.Errorf("Unexpected error during shutdown: %v", err)
	}
}

func TestFrameworkComponentsGetters(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	pool := NewWorkerPool(5, 100)
	fetcher := DataFetcherFunc(func() ([]interface{}, error) {
		return []interface{}{}, nil
	})
	poller := NewPoller(DefaultPollerConfig(), pool, fetcher)

	components := &FrameworkComponents{
		Config:     config,
		DB:         nil,
		WorkerPool: pool,
		Poller:     poller,
	}

	if components.Config != config {
		t.Error("Expected Config to return config")
	}
	if components.DB != nil {
		t.Error("Expected DB to be nil")
	}
	if components.WorkerPool != pool {
		t.Error("Expected WorkerPool to return pool")
	}
	if components.Poller != poller {
		t.Error("Expected Poller to return poller")
	}
}
