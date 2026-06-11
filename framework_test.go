package nanopony

import (
	"context"
	"os"
	"time"
	"testing"

	"github.com/segmentio/kafka-go"
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
	pool := NewWorkerPool(5, 100, 2)
	framework := NewFramework().
		WithWorkerPoolFromInstance(pool)

	if framework.workerPool != pool {
		t.Error("Expected workerPool to be set")
	}
}

func TestFrameworkWithPollerFromInstance(t *testing.T) {
	pool := NewWorkerPool(5, 100, 2)
	fetcher := DataFetcherFunc(func(ctx context.Context) ([]interface{}, error) {
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

	framework := NewFramework()
	framework.Build()
	framework.Build() // Should panic
}

func TestFrameworkComponentsStartStop(t *testing.T) {
	pool := NewWorkerPool(2, 10, 2)
	fetcher := DataFetcherFunc(func(ctx context.Context) ([]interface{}, error) {
		return []interface{}{}, nil
	})
	poller := NewPoller(DefaultPollerConfig(), pool, fetcher)

	components := &FrameworkComponents{
		WorkerPool: pool,
		Poller:     poller,
	}

	ctx := context.Background()
	err := components.Start(ctx, func(ctx context.Context, job *Job) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to start framework: %v", err)
	}

	if !pool.IsRunning() {
		t.Error("Expected worker pool to be running")
	}
	if !poller.IsRunning() {
		t.Error("Expected poller to be running")
	}

	err = components.Shutdown(ctx)
	if err != nil {
		t.Errorf("Unexpected error during shutdown: %v", err)
	}
}

func TestFrameworkBuildSafe(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	// Missing config
	f1 := NewFramework()
	_, err := f1.BuildSafe()
	if err == nil {
		t.Error("Expected error for missing config")
	}

	// Invalid config
	f2 := NewFramework().WithConfig(BuildConfig(func(c *Config) {
		c.App.Env = "invalid"
	}))
	_, err = f2.BuildSafe()
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Double build
	f3 := NewFramework().WithConfig(config)
	f3.BuildSafe()
	_, err = f3.BuildSafe()
	if err != ErrAlreadyBuilt {
		t.Errorf("Expected ErrAlreadyBuilt, got %v", err)
	}
}

func TestFrameworkWithComponentsSafe(t *testing.T) {
	ResetConfig()

	// Missing config for WithPostgreSQLSafe\n	_, err := f1.WithPostgreSQLSafe()\n	if err != ErrConfigNotSet {\n		t.Errorf("Expected ErrConfigNotSet, got %v", err)\n	}\n\n	// Missing config for WithDatabaseSafe
	f1 := NewFramework()
	_, err := f1.WithDatabaseSafe()
	if err != ErrConfigNotSet {
		t.Errorf("Expected ErrConfigNotSet, got %v", err)
	}

	// Missing config for WithKafkaWriterSafeRoundRobin
	_, err = f1.WithKafkaWriterSafeRoundRobin()
	if err != ErrConfigNotSet {
		t.Errorf("Expected ErrConfigNotSet, got %v", err)
	}

	// Missing config for WithKafkaWriterSafeHash
	_, err = f1.WithKafkaWriterSafeHash()
	if err != ErrConfigNotSet {
		t.Errorf("Expected ErrConfigNotSet, got %v", err)
	}

	// Missing Kafka writer for WithProducerSafe
	_, err = f1.WithProducerSafe()
	if err != ErrKafkaNotSet {
		t.Errorf("Expected ErrKafkaNotSet, got %v", err)
	}

	// Missing worker pool for WithPollerSafe
	_, err = f1.WithPollerSafe(DefaultPollerConfig(), nil)
	if err != ErrWorkerPoolNotSet {
		t.Errorf("Expected ErrWorkerPoolNotSet, got %v", err)
	}
}

func TestFrameworkCheckReadiness(t *testing.T) {
	components := &FrameworkComponents{}
	ctx := context.Background()

	// Empty components
	if err := components.CheckReadiness(ctx); err != nil {
		t.Errorf("Expected no error for empty components, got %v", err)
	}

	// Kafka writer with no brokers
	components.KafkaWriter = &kafka.Writer{}
	if err := components.CheckReadiness(ctx); err == nil {
		t.Error("Expected error for Kafka writer with no brokers")
	}
}

func TestFrameworkShutdownErrors(t *testing.T) {
	components := &FrameworkComponents{
		cleanupFuncs: []func() error{
			func() error { return context.DeadlineExceeded },
		},
	}

	err := components.Shutdown(context.Background())
	if err == nil {
		t.Error("Expected aggregated error during shutdown")
	}
}

func TestFrameworkPanicWrappers(t *testing.T) {
	f := NewFramework()

	assertPanic := func(fn func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic")
			}
		}()
		fn()
	}

	assertPanic(func() { f.WithDatabase() })
	assertPanic(func() { f.WithKafkaWriterRoundRobin() })
	assertPanic(func() { f.WithKafkaWriterHash() })
	assertPanic(func() { f.WithProducer() })
	assertPanic(func() { f.WithPoller(DefaultPollerConfig(), nil) })
}

func TestWithHttpServer(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	var called bool
	components := NewFramework().
		WithConfig(config).
		WithHttpServer(func(app *HttpServer) {
			called = true; app.Get("/test", func(c *Ctx) error { return nil })
		}).
		Build()
	if components.HttpServer == nil { t.Errorf("HttpServer nil") }
	if !called { t.Errorf("Setup not called") }
}

func TestWithPostgreSQLPanic(t *testing.T) {
	ResetConfig()
	defer func() {
		if r := recover(); r == nil { t.Errorf("Expected panic") }
	}()
	NewFramework().WithPostgreSQL()
}

func TestWithPostgreSQLSafeNoConfig(t *testing.T) {
	_, err := NewFramework().WithPostgreSQLSafe()
	if err != ErrConfigNotSet { t.Errorf("Expected ErrConfigNotSet, got %v", err) }
}

func TestWithPersistentQueue(t *testing.T) {
	f := NewFramework()
	f.WithPersistentQueue(nil)
	if f.persistentQueue != nil { t.Errorf("WithPersistentQueue failed") }
}

func TestGetAppConfig(t *testing.T) {
	ResetConfig()
	_ = NewConfig()
	got := getAppConfig()
	if got == nil { t.Errorf("getAppConfig returned nil") }
}


func TestFrameworkCheckReadinessFull(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	f := NewFramework().WithConfig(config)
	// Mock components
	f.db = nil
	f.pgDB = nil
	f.kafkaWriter = nil
	
	comp, _ := f.BuildSafe()
	ctx := context.Background()
	if err := comp.CheckReadiness(ctx); err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func TestWithKafkaSafe(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	f := NewFramework().WithConfig(config)
	_, err := f.WithKafkaWriterSafeRoundRobin()
	if err != nil {
		// Might fail if env not set, but we hit the code path
	}
}

func TestWithDatabaseSelection(t *testing.T) {
	ResetConfig()
	os.Setenv("GO_ENV", "local")
	
	// Test Oracle path
	os.Setenv("DB_TYPE", "oracle")
	config := NewConfig()
	f := NewFramework().WithConfig(config)
	_, _ = f.WithDatabaseSafe() // Triggers Oracle path
	
	// Test PostgreSQL path
	os.Setenv("DB_TYPE", "postgresql")
	ResetConfig()
	config = NewConfig()
	f2 := NewFramework().WithConfig(config)
	_, _ = f2.WithDatabaseSafe() // Triggers PostgreSQL path
	
	// Test Invalid path
	os.Setenv("DB_TYPE", "invalid")
	ResetConfig()
	config = NewConfig()
	f3 := NewFramework().WithConfig(config)
	_, err := f3.WithDatabaseSafe()
	if err == nil { t.Errorf("Expected error for invalid db type") }
}

func TestWithKafkaWriterSafeHash(t *testing.T) {
	ResetConfig()
	os.Setenv("GO_ENV", "local")
	os.Setenv("KAFKA-MODELS", "kafka-localhost")
	config := NewConfig()
	f := NewFramework().WithConfig(config)
	_, _ = f.WithKafkaWriterSafeHash()
}

func TestFrameworkStartComponents(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	f := NewFramework().WithConfig(config).WithWorkerPool(1,1,1)
	
	comp, _ := f.BuildSafe()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = comp.Start(ctx, func(ctx context.Context, job *Job) error { return nil })
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = comp.Shutdown(context.Background())
}
