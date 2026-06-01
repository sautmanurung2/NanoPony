// framework.go — Builder pattern for wiring NanoPony components together.
package nanopony

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Framework errors
var (
	ErrQueueFull        = errors.New("job queue is full")
	ErrConfigNotSet     = errors.New("config must be set before this operation")
	ErrDatabaseNotSet   = errors.New("database must be set before this operation")
	ErrKafkaNotSet      = errors.New("kafka writer must be set before this operation")
	ErrWorkerPoolNotSet = errors.New("worker pool must be set before this operation")
	ErrAlreadyBuilt     = errors.New("framework has already been built")
)

// Framework is the main builder for setting up the Kafka-Oracle framework.
type Framework[T any] struct {
	config       *Config
	db           *sql.DB
	kafkaWriter  *kafka.Writer
	producer     *KafkaProducer
	workerPool   *ShardedWorkerPool[T]
	poller       *Poller[T]
	cleanupFuncs []func() error
	built        bool
}

// NewFramework creates a new framework builder with default values.
func NewFramework[T any]() *Framework[T] {
	return &Framework[T]{
		cleanupFuncs: make([]func() error, 0),
	}
}

// WithConfig sets the configuration for the framework.
func (f *Framework[T]) WithConfig(config *Config) *Framework[T] {
	f.config = config
	return f
}

// WithDatabase sets up the Oracle database connection.
func (f *Framework[T]) WithDatabase() *Framework[T] {
	fw, err := f.WithDatabaseSafe()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithDatabaseSafe sets up the Oracle database connection with error handling.
func (f *Framework[T]) WithDatabaseSafe() (*Framework[T], error) {
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

// WithDatabaseFromInstance allows using an existing database connection.
func (f *Framework[T]) WithDatabaseFromInstance(db *sql.DB) *Framework[T] {
	f.db = db
	return f
}

// WithKafkaWriterRoundRobin sets up the Kafka writer.
func (f *Framework[T]) WithKafkaWriterRoundRobin() *Framework[T] {
	fw, err := f.WithKafkaWriterSafeRoundRobin()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithKafkaWriterHash sets up the Kafka writer with Hash balancer.
func (f *Framework[T]) WithKafkaWriterHash() *Framework[T] {
	fw, err := f.WithKafkaWriterSafeHash()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithKafkaWriterSafeHash sets up the Kafka writer with Hash balancer with error handling.
func (f *Framework[T]) WithKafkaWriterSafeHash() (*Framework[T], error) {
	if f.config == nil {
		return f, ErrConfigNotSet
	}

	f.kafkaWriter = NewKafkaWriterFromConfigHash(f.config)
	f.AddCleanup(func() error {
		return CloseKafkaWriter(f.kafkaWriter)
	})

	return f, nil
}

// WithKafkaWriterSafeRoundRobin sets up the Kafka writer with error handling.
func (f *Framework[T]) WithKafkaWriterSafeRoundRobin() (*Framework[T], error) {
	if f.config == nil {
		return f, ErrConfigNotSet
	}

	f.kafkaWriter = NewKafkaWriterFromConfigRoundRobin(f.config)
	f.AddCleanup(func() error {
		return CloseKafkaWriter(f.kafkaWriter)
	})

	return f, nil
}

// WithKafkaWriterFromInstance allows using an existing Kafka writer.
func (f *Framework[T]) WithKafkaWriterFromInstance(writer *kafka.Writer) *Framework[T] {
	f.kafkaWriter = writer
	return f
}

// WithProducer sets up the Kafka producer.
func (f *Framework[T]) WithProducer() *Framework[T] {
	fw, err := f.WithProducerSafe()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithProducerSafe sets up the Kafka producer with error handling.
func (f *Framework[T]) WithProducerSafe() (*Framework[T], error) {
	if f.kafkaWriter == nil {
		return f, ErrKafkaNotSet
	}

	f.producer = NewKafkaProducer(f.kafkaWriter)
	return f, nil
}

// WithProducerFromInstance allows using an existing producer instance.
func (f *Framework[T]) WithProducerFromInstance(producer *KafkaProducer) *Framework[T] {
	f.producer = producer
	return f
}

// WithWorkerPool sets up the worker pool.
func (f *Framework[T]) WithWorkerPool(numWorkers, queueSize, numShards int) *Framework[T] {
	f.workerPool = NewWorkerPool[T](numWorkers, queueSize, numShards)
	return f
}

// WithWorkerPoolFromInstance allows using an existing worker pool.
func (f *Framework[T]) WithWorkerPoolFromInstance(pool *ShardedWorkerPool[T]) *Framework[T] {
	f.workerPool = pool
	return f
}

// WithPoller sets up the poller.
func (f *Framework[T]) WithPoller(config PollerConfig, dataFetcher DataFetcher[T]) *Framework[T] {
	fw, err := f.WithPollerSafe(config, dataFetcher)
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithPollerSafe sets up the poller with error handling.
func (f *Framework[T]) WithPollerSafe(config PollerConfig, dataFetcher DataFetcher[T]) (*Framework[T], error) {
	if f.workerPool == nil {
		return f, ErrWorkerPoolNotSet
	}

	f.poller = NewPoller[T](config, f.workerPool, dataFetcher)
	return f, nil
}

// WithPollerFromInstance allows using an existing poller instance.
func (f *Framework[T]) WithPollerFromInstance(poller *Poller[T]) *Framework[T] {
	f.poller = poller
	return f
}

// AddCleanup adds a cleanup function to be called during shutdown.
func (f *Framework[T]) AddCleanup(fn func() error) *Framework[T] {
	f.cleanupFuncs = append(f.cleanupFuncs, fn)
	return f
}

// Build finalizes the configuration and returns FrameworkComponents.
func (f *Framework[T]) Build() *FrameworkComponents[T] {
	components, err := f.BuildSafe()
	if err != nil {
		panic(err.Error())
	}
	return components
}

// BuildSafe finalizes the configuration and returns FrameworkComponents.
func (f *Framework[T]) BuildSafe() (*FrameworkComponents[T], error) {
	if f.built {
		return nil, ErrAlreadyBuilt
	}

	if f.config == nil {
		return nil, fmt.Errorf("Config is required. Call WithConfig() before Build()")
	}

	if err := f.config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	f.built = true

	if f.kafkaWriter != nil && f.workerPool != nil {
		f.kafkaWriter.BatchSize = f.workerPool.NumWorkers()
	}

	return &FrameworkComponents[T]{
		Config:       f.config,
		DB:           f.db,
		KafkaWriter:  f.kafkaWriter,
		Producer:     f.producer,
		WorkerPool:   f.workerPool,
		Poller:       f.poller,
		cleanupFuncs: f.cleanupFuncs,
	}, nil
}

// FrameworkComponents holds all initialized framework components.
type FrameworkComponents[T any] struct {
	Config       *Config
	DB           *sql.DB
	KafkaWriter  *kafka.Writer
	Producer     *KafkaProducer
	WorkerPool   *ShardedWorkerPool[T]
	Poller       *Poller[T]
	cleanupFuncs []func() error
}

// Start starts all framework components.
func (fc *FrameworkComponents[T]) Start(ctx context.Context, handler JobHandler[T]) {
	if fc.WorkerPool != nil {
		fc.WorkerPool.Start(ctx, handler)
	}

	if fc.Poller != nil {
		fc.Poller.Start()
	}
}

// Shutdown gracefully shuts down all framework components.
func (fc *FrameworkComponents[T]) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var allErrors []error

	collectErr := func(err error) {
		if err != nil {
			errMu.Lock()
			allErrors = append(allErrors, err)
			errMu.Unlock()
		}
	}

	if fc.Poller != nil {
		fc.Poller.Stop()
	}

	if fc.WorkerPool != nil {
		fc.WorkerPool.Stop()
	}

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

	return errors.Join(allErrors...)
}
