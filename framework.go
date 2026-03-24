package nanopony

import (
	"context"
	"database/sql"
	"errors"
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

// Framework is the main builder for setting up the Kafka-Oracle framework
type Framework struct {
	config       *Config
	db           *sql.DB
	kafkaWriter  *kafka.Writer
	producer     *KafkaProducer
	workerPool   *WorkerPool
	poller       *Poller
	repositories []Repository
	services     []Service
	cleanupFuncs []func() error
	built        bool
}

// NewFramework creates a new framework builder
func NewFramework() *Framework {
	return &Framework{
		repositories: make([]Repository, 0),
		services:     make([]Service, 0),
		cleanupFuncs: make([]func() error, 0),
	}
}

// WithConfig sets the configuration
func (f *Framework) WithConfig(config *Config) *Framework {
	f.config = config
	return f
}

// WithDatabase sets up the Oracle database connection
func (f *Framework) WithDatabase() *Framework {
	if f.config == nil {
		panic(ErrConfigNotSet.Error())
	}

	db, err := NewOracleFromConfig(f.config)
	if err != nil {
		panic(err)
	}

	f.db = db
	f.AddCleanup(func() error {
		return CloseDB(f.db)
	})

	return f
}

// WithDatabaseFromConnection allows using an existing database connection
func (f *Framework) WithDatabaseFromConnection(db *sql.DB) *Framework {
	f.db = db
	return f
}

// WithKafkaWriter sets up the Kafka writer
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
func (f *Framework) WithKafkaWriterFromInstance(writer *kafka.Writer) *Framework {
	f.kafkaWriter = writer
	return f
}

// WithProducer sets up the Kafka producer
func (f *Framework) WithProducer() *Framework {
	if f.kafkaWriter == nil {
		panic(ErrKafkaNotSet.Error())
	}

	f.producer = NewKafkaProducer(f.kafkaWriter)
	return f
}

// WithProducerFromInstance allows using an existing producer instance
func (f *Framework) WithProducerFromInstance(producer *KafkaProducer) *Framework {
	f.producer = producer
	return f
}

// WithWorkerPool sets up the worker pool
func (f *Framework) WithWorkerPool(numWorkers, queueSize int) *Framework {
	f.workerPool = NewWorkerPool(numWorkers, queueSize)
	return f
}

// WithWorkerPoolFromInstance allows using an existing worker pool
func (f *Framework) WithWorkerPoolFromInstance(pool *WorkerPool) *Framework {
	f.workerPool = pool
	return f
}

// WithPoller sets up the poller
func (f *Framework) WithPoller(config PollerConfig, dataFetcher DataFetcher) *Framework {
	if f.workerPool == nil {
		panic(ErrWorkerPoolNotSet.Error())
	}

	f.poller = NewPoller(config, f.workerPool, dataFetcher)
	return f
}

// WithPollerFromInstance allows using an existing poller instance
func (f *Framework) WithPollerFromInstance(poller *Poller) *Framework {
	f.poller = poller
	return f
}

// AddRepository adds a repository to the framework
func (f *Framework) AddRepository(repo Repository) *Framework {
	f.repositories = append(f.repositories, repo)
	return f
}

// AddService adds a service to the framework
func (f *Framework) AddService(service Service) *Framework {
	f.services = append(f.services, service)
	return f
}

// AddCleanup adds a cleanup function to be called on shutdown
func (f *Framework) AddCleanup(fn func() error) *Framework {
	f.cleanupFuncs = append(f.cleanupFuncs, fn)
	return f
}

// Build builds and returns the framework components
func (f *Framework) Build() *FrameworkComponents {
	if f.built {
		panic(ErrAlreadyBuilt.Error())
	}
	f.built = true

	return &FrameworkComponents{
		Config:       f.config,
		DB:           f.db,
		KafkaWriter:  f.kafkaWriter,
		Producer:     f.producer,
		WorkerPool:   f.workerPool,
		Poller:       f.poller,
		repositories: f.repositories,
		services:     f.services,
		cleanupFuncs: f.cleanupFuncs,
	}
}

// FrameworkComponents holds all initialized framework components
type FrameworkComponents struct {
	Config       *Config
	DB           *sql.DB
	KafkaWriter  *kafka.Writer
	Producer     *KafkaProducer
	WorkerPool   *WorkerPool
	Poller       *Poller
	repositories []Repository
	services     []Service
	cleanupFuncs []func() error
}

// Start starts all framework components
func (fc *FrameworkComponents) Start(ctx context.Context, handler JobHandler) {
	if fc.WorkerPool != nil {
		fc.WorkerPool.Start(ctx, handler)
	}

	if fc.Poller != nil {
		fc.Poller.Start()
	}

	// Initialize all services
	for _, service := range fc.services {
		if err := service.Initialize(); err != nil {
			// Log error but continue
			// In production, use proper logging
		}
	}
}

// Shutdown gracefully shuts down all framework components
func (fc *FrameworkComponents) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	// Stop poller first
	if fc.Poller != nil {
		fc.Poller.Stop()
	}

	// Stop worker pool
	if fc.WorkerPool != nil {
		fc.WorkerPool.Stop()
	}

	// Shutdown all services
	for _, service := range fc.services {
		if err := service.Shutdown(); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMu.Unlock()
		}
	}

	// Close all repositories
	for _, repo := range fc.repositories {
		if err := repo.Close(); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMu.Unlock()
		}
	}

	// Run cleanup functions
	for _, cleanup := range fc.cleanupFuncs {
		wg.Add(1)
		go func(fn func() error) {
			defer wg.Done()
			if err := fn(); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
		}(cleanup)
	}

	wg.Wait()

	return firstErr
}

// GetDB returns the database connection
func (fc *FrameworkComponents) GetDB() *sql.DB {
	return fc.DB
}

// GetProducer returns the Kafka producer
func (fc *FrameworkComponents) GetProducer() *KafkaProducer {
	return fc.Producer
}

// GetConfig returns the configuration
func (fc *FrameworkComponents) GetConfig() *Config {
	return fc.Config
}

// GetWorkerPool returns the worker pool
func (fc *FrameworkComponents) GetWorkerPool() *WorkerPool {
	return fc.WorkerPool
}

// GetPoller returns the poller
func (fc *FrameworkComponents) GetPoller() *Poller {
	return fc.Poller
}
