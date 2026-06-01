package nanopony

import (
	"context"
	"testing"
)

func TestNewFramework(t *testing.T) {
	framework := NewFramework[any]()
	if framework == nil {
		t.Fatal("Expected framework to be created")
	}
}

func TestFrameworkBuilder(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	framework := NewFramework[any]().
		WithConfig(config)

	if framework.config != config {
		t.Error("Expected config to be set")
	}
}

func TestFrameworkWithDatabaseFromInstance(t *testing.T) {
	framework := NewFramework[any]().
		WithDatabaseFromInstance(nil)

	if framework.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestFrameworkWithKafkaWriterFromInstance(t *testing.T) {
	framework := NewFramework[any]().
		WithKafkaWriterFromInstance(nil)

	if framework.kafkaWriter != nil {
		t.Error("Expected kafkaWriter to be nil")
	}
}

func TestFrameworkWithProducerFromInstance(t *testing.T) {
	framework := NewFramework[any]().
		WithProducerFromInstance(nil)

	if framework.producer != nil {
		t.Error("Expected producer to be nil")
	}
}

func TestFrameworkWithWorkerPoolFromInstance(t *testing.T) {
	pool := NewWorkerPool[any](5, 100, 2)
	framework := NewFramework[any]().
		WithWorkerPoolFromInstance(pool)

	if framework.workerPool != pool {
		t.Error("Expected workerPool to be set")
	}
}

func TestFrameworkWithPollerFromInstance(t *testing.T) {
	pool := NewWorkerPool[any](5, 100, 2)
	fetcher := DataFetcherFunc[any](func() ([]any, error) {
		return []any{}, nil
	})
	poller := NewPoller[any](DefaultPollerConfig(), pool, fetcher)

	framework := NewFramework[any]().
		WithPollerFromInstance(poller)

	if framework.poller != poller {
		t.Error("Expected poller to be set")
	}
}

func TestFrameworkAddCleanup(t *testing.T) {
	cleanupCalled := false
	framework := NewFramework[any]().
		AddCleanup(func() error {
			cleanupCalled = true
			return nil
		})

	if len(framework.cleanupFuncs) != 1 {
		t.Errorf("Expected 1 cleanup function, got %d", len(framework.cleanupFuncs))
	}

	framework.cleanupFuncs[0]()
	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}
}

func TestFrameworkBuild(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	framework := NewFramework[any]().
		WithConfig(config).
		WithWorkerPool(5, 100, 2)

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

	framework := NewFramework[any]()
	framework.Build()
	framework.Build()
}

func TestFrameworkComponentsStartStop(t *testing.T) {
	pool := NewWorkerPool[any](2, 10, 2)
	fetcher := DataFetcherFunc[any](func() ([]any, error) {
		return []any{}, nil
	})
	poller := NewPoller[any](DefaultPollerConfig(), pool, fetcher)

	components := &FrameworkComponents[any]{
		WorkerPool: pool,
		Poller:     poller,
	}

	ctx := context.Background()
	components.Start(ctx, func(ctx context.Context, job *Job[any]) error {
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
	pool := NewWorkerPool[any](5, 100, 2)
	fetcher := DataFetcherFunc[any](func() ([]any, error) {
		return []any{}, nil
	})
	poller := NewPoller[any](DefaultPollerConfig(), pool, fetcher)

	components := &FrameworkComponents[any]{
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
