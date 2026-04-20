package nanopony

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Framework errors
var (
	// ErrQueueFull is returned when the job queue is full and cannot accept new jobs
	ErrQueueFull = errors.New("job queue is full")
	// ErrConfigNotSet is returned when config is required but not set
	ErrConfigNotSet = errors.New("config must be set before this operation")
	// ErrDatabaseNotSet is returned when database is required but not set
	ErrDatabaseNotSet = errors.New("database must be set before this operation")
	// ErrKafkaNotSet is returned when kafka writer is required but not set
	ErrKafkaNotSet = errors.New("kafka writer must be set before this operation")
	// ErrWorkerPoolNotSet is returned when worker pool is required but not set
	ErrWorkerPoolNotSet = errors.New("worker pool must be set before this operation")
	// ErrAlreadyBuilt is returned when Build() is called more than once
	ErrAlreadyBuilt = errors.New("framework has already been built")
)

// Framework is the main builder for setting up the Kafka-Oracle framework.
// It uses the builder pattern to configure and wire together components
// such as database connections, Kafka producers, worker pools, and pollers.
//
// Example usage:
//
//	fw := nanopony.NewFramework().
//	    WithConfig(config).
//	    WithDatabase().
//	    WithKafkaWriter().
//	    WithProducer().
//	    WithWorkerPool(5, 100).
//	    WithPoller(pollerConfig, dataFetcher).
//	    Build()
//
//	fw.Start(ctx, jobHandler)
//	defer fw.Shutdown(ctx)
type Framework struct {
	config       *Config
	db           *sql.DB
	kafkaWriter  *kafka.Writer
	producer     *KafkaProducer
	workerPool   *WorkerPool
	poller       *Poller
	cleanupFuncs []func() error
	built        bool
}

// NewFramework creates a new framework builder with default values.
// Use the With* methods to configure components before calling Build().
func NewFramework() *Framework {
	return &Framework{
		cleanupFuncs: make([]func() error, 0),
	}
}

// WithConfig sets the configuration for the framework.
// This should be called before other With* methods that depend on config.
func (f *Framework) WithConfig(config *Config) *Framework {
	f.config = config
	return f
}

// WithDatabase sets up the Oracle database connection using the configured config.
// Requires WithConfig to be called first.
//
// Panics if config is not set or connection fails.
// For error-handling version, see WithDatabaseSafe.
func (f *Framework) WithDatabase() *Framework {
	if f.config == nil {
		panic(ErrConfigNotSet.Error())
	}

	db, err := NewOracleFromConfig(f.config)
	if err != nil {
		panic(fmt.Errorf("failed to create database connection: %w", err))
	}

	f.db = db
	f.AddCleanup(func() error {
		return CloseDB(f.db)
	})

	return f
}

// WithDatabaseSafe sets up the Oracle database connection with error handling.
// Unlike WithDatabase, this method returns an error instead of panicking.
// This is recommended for production code.
func (f *Framework) WithDatabaseSafe() (*Framework, error) {
	if f.config == nil {
		return f, ErrConfigNotSet
	}

	db, err := NewOracleFromConfig(f.config)
	if err != nil {
		return f, fmt.Errorf("failed to create database connection: %w", err)
	}

	f.db = db
	f.AddCleanup(func() error {
		return CloseDB(f.db)
	})

	return f, nil
}

// WithDatabaseFromInstance allows using an existing database connection
// instead of creating a new one from config.
func (f *Framework) WithDatabaseFromInstance(db *sql.DB) *Framework {
	f.db = db
	return f
}

// WithKafkaWriter sets up the Kafka writer using the configured config.
// Requires WithConfig to be called first.
//
// Panics if config is not set.
func (f *Framework) WithKafkaWriter() *Framework {
	if f.config == nil {
		panic(ErrConfigNotSet.Error())
	}

	f.kafkaWriter = NewKafkaWriterFromConfig(f.config)
	f.AddCleanup(func() error {
		return CloseKafkaWriter(f.kafkaWriter)
	})

	return f
}

// WithKafkaWriterFromInstance allows using an existing Kafka writer
// instead of creating a new one from config.
func (f *Framework) WithKafkaWriterFromInstance(writer *kafka.Writer) *Framework {
	f.kafkaWriter = writer
	return f
}

// WithProducer sets up the Kafka producer using the configured Kafka writer.
// Requires WithKafkaWriter or WithKafkaWriterFromInstance to be called first.
//
// Panics if kafka writer is not set.
func (f *Framework) WithProducer() *Framework {
	if f.kafkaWriter == nil {
		panic(ErrKafkaNotSet.Error())
	}

	f.producer = NewKafkaProducer(f.kafkaWriter)
	return f
}

// WithProducerFromInstance allows using an existing producer instance
// instead of creating a new one.
func (f *Framework) WithProducerFromInstance(producer *KafkaProducer) *Framework {
	f.producer = producer
	return f
}

// WithWorkerPool sets up the worker pool with the specified number of workers and queue size.
// Example: WithWorkerPool(5, 100) creates 5 workers with a queue size of 100.
func (f *Framework) WithWorkerPool(numWorkers, queueSize int) *Framework {
	f.workerPool = NewWorkerPool(numWorkers, queueSize)
	return f
}

// WithWorkerPoolFromInstance allows using an existing worker pool
// instead of creating a new one.
func (f *Framework) WithWorkerPoolFromInstance(pool *WorkerPool) *Framework {
	f.workerPool = pool
	return f
}

// WithPoller sets up the poller with the given configuration and data fetcher.
// Requires WithWorkerPool to be called first.
//
// The poller periodically fetches data and submits it as jobs to the worker pool.
//
// Panics if worker pool is not set.
func (f *Framework) WithPoller(config PollerConfig, dataFetcher DataFetcher) *Framework {
	if f.workerPool == nil {
		panic(ErrWorkerPoolNotSet.Error())
	}

	f.poller = NewPoller(config, f.workerPool, dataFetcher)
	return f
}

// WithPollerFromInstance allows using an existing poller instance
// instead of creating a new one.
func (f *Framework) WithPollerFromInstance(poller *Poller) *Framework {
	f.poller = poller
	return f
}

// AddCleanup adds a cleanup function to be called during shutdown.
// Cleanup functions are executed concurrently.
func (f *Framework) AddCleanup(fn func() error) *Framework {
	f.cleanupFuncs = append(f.cleanupFuncs, fn)
	return f
}

// Build finalizes the configuration and returns FrameworkComponents.
// This method should be called after all With* methods.
//
// Panics:
//   - if Build() is called more than once
//   - if Config is missing (WithConfig was not called)
//   - if Config validation fails (e.g., missing environment variables)
func (f *Framework) Build() *FrameworkComponents {
	if f.built {
		panic(ErrAlreadyBuilt.Error())
	}

	if f.config == nil {
		panic("Config is required. Call WithConfig() before Build().")
	}

	// Perform fail-fast validation of environment configuration
	if err := f.config.Validate(); err != nil {
		panic(fmt.Sprintf("Configuration validation failed: %v", err))
	}

	f.built = true

	// Dynamically adjust Kafka BatchSize based on worker pool size
	if f.kafkaWriter != nil && f.workerPool != nil {
		f.kafkaWriter.BatchSize = f.workerPool.numWorkers
	}

	return &FrameworkComponents{
		Config:       f.config,
		DB:           f.db,
		KafkaWriter:  f.kafkaWriter,
		Producer:     f.producer,
		WorkerPool:   f.workerPool,
		Poller:       f.poller,
		cleanupFuncs: f.cleanupFuncs,
	}
}

// FrameworkComponents holds all initialized framework components.
// Use this struct to access components like DB, Producer, WorkerPool, etc.
type FrameworkComponents struct {
	// Config holds the application configuration
	Config *Config
	// DB is the Oracle database connection
	DB *sql.DB
	// KafkaWriter is the Kafka writer for producing messages
	KafkaWriter *kafka.Writer
	// Producer is the Kafka producer wrapper
	Producer *KafkaProducer
	// WorkerPool manages the concurrent job processing
	WorkerPool *WorkerPool
	// Poller periodically fetches data and submits jobs
	Poller *Poller

	cleanupFuncs []func() error
}

// Start starts all framework components.
// It starts the worker pool with the given handler, starts the poller,
// and initializes all services.
//
// Note: Service initialization errors are logged but do not prevent
// the framework from starting. This is intentional to allow partial failures.
func (fc *FrameworkComponents) Start(ctx context.Context, handler JobHandler) {
	// Start worker pool
	if fc.WorkerPool != nil {
		fc.WorkerPool.Start(ctx, handler)
	}

	// Start poller
	if fc.Poller != nil {
		fc.Poller.Start()
	}
}

// Shutdown gracefully shuts down all framework components in the following order:
// 1. Stop poller
// 2. Stop worker pool
// 3. Shutdown all services
// 4. Close all repositories
// 5. Run all cleanup functions concurrently
//
// If multiple errors occur, they are all collected and returned as an aggregated error.
// Returns nil if all shutdowns succeeded.
func (fc *FrameworkComponents) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var allErrors []error

	// Helper function to collect errors safely
	collectErr := func(err error) {
		if err != nil {
			errMu.Lock()
			allErrors = append(allErrors, err)
			errMu.Unlock()
		}
	}

	// Stop poller first
	if fc.Poller != nil {
		fc.Poller.Stop()
	}

	// Stop worker pool
	if fc.WorkerPool != nil {
		fc.WorkerPool.Stop()
	}

	// Run cleanup functions concurrently
	for _, cleanup := range fc.cleanupFuncs {
		wg.Add(1)
		go func(fn func() error) {
			defer wg.Done()
			if err := fn(); err != nil {
				collectErr(fmt.Errorf("cleanup failed: %w", err))
			}
		}(cleanup)
	}

	wg.Wait()

	// Return aggregated errors or nil
	if len(allErrors) == 0 {
		return nil
	}
	if len(allErrors) == 1 {
		return allErrors[0]
	}

	// Join all errors for comprehensive error reporting
	var errMsgs []string
	for _, err := range allErrors {
		errMsgs = append(errMsgs, err.Error())
	}
	return errors.New("multiple shutdown errors: " + strings.Join(errMsgs, "; "))
}

// GetDB returns the database connection.
// Returns nil if database was not configured.
// Use HasDB() to check before use.
func (fc *FrameworkComponents) GetDB() *sql.DB {
	return fc.DB
}

// HasDB returns true if a database connection was configured.
func (fc *FrameworkComponents) HasDB() bool {
	return fc.DB != nil
}

// GetProducer returns the Kafka producer.
// Returns nil if producer was not configured.
// Use HasProducer() to check before use.
func (fc *FrameworkComponents) GetProducer() *KafkaProducer {
	return fc.Producer
}

// HasProducer returns true if a Kafka producer was configured.
func (fc *FrameworkComponents) HasProducer() bool {
	return fc.Producer != nil
}

// GetConfig returns the configuration.
func (fc *FrameworkComponents) GetConfig() *Config {
	return fc.Config
}

// HasConfig returns true if configuration was set.
func (fc *FrameworkComponents) HasConfig() bool {
	return fc.Config != nil
}

// GetWorkerPool returns the worker pool.
// Returns nil if worker pool was not configured.
// Use HasWorkerPool() to check before use.
func (fc *FrameworkComponents) GetWorkerPool() *WorkerPool {
	return fc.WorkerPool
}

// HasWorkerPool returns true if a worker pool was configured.
func (fc *FrameworkComponents) HasWorkerPool() bool {
	return fc.WorkerPool != nil
}

// GetPoller returns the poller.
// Returns nil if poller was not configured.
// Use HasPoller() to check before use.
func (fc *FrameworkComponents) GetPoller() *Poller {
	return fc.Poller
}

// HasPoller returns true if a poller was configured.
func (fc *FrameworkComponents) HasPoller() bool {
	return fc.Poller != nil
}
