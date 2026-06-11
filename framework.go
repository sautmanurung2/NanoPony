// framework.go — Builder pattern for wiring NanoPony components together.
package nanopony

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	ErrPoolStopped      = errors.New("worker pool is stopped")
	ErrAlreadyBuilt     = errors.New("framework has already been built")
)

// Framework is the main builder for setting up the Kafka-Oracle framework.
type Framework struct {
	config          *Config
	db              *sql.DB
	pgDB            *sql.DB
	kafkaWriter     *kafka.Writer
	producer        *KafkaProducer
	workerPool      *ShardedWorkerPool
	poller          *Poller
	httpServer      *HttpServer
	cleanupFuncs    []func() error
	built           bool
	persistentQueue PersistentQueue
}

// NewFramework creates a new framework builder.
func NewFramework() *Framework {
	return &Framework{
		cleanupFuncs: make([]func() error, 0),
	}
}

// WithConfig sets the configuration for the framework.
func (f *Framework) WithConfig(config *Config) *Framework {
	f.config = config
	return f
}

// WithDatabase sets up the Oracle database connection.
func (f *Framework) WithDatabase() *Framework {
	fw, err := f.WithDatabaseSafe()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithDatabaseSafe sets up the database connection (Oracle or PostgreSQL) based on config.
func (f *Framework) WithDatabaseSafe() (*Framework, error) {
	if f.config == nil {
		return f, ErrConfigNotSet
	}

	dbType := f.config.App.DBType
	switch dbType {
	case "", "oracle":
		db, err := NewOracleFromConfig(f.config)
		if err != nil {
			return f, fmt.Errorf("failed to create Oracle connection: %w", err)
		}
		f.db = db
		f.AddCleanup(func() error { return CloseDB(f.db) })
	case "postgresql":
		db, err := NewPostgreSQLFromConfig(f.config)
		if err != nil {
			return f, fmt.Errorf("failed to create PostgreSQL connection: %w", err)
		}
		f.pgDB = db
		f.AddCleanup(func() error { return CloseDB(f.pgDB) })
	default:
		return f, fmt.Errorf("unsupported database type: %s", dbType)
	}

	return f, nil
}

// WithPostgreSQL sets up the PostgreSQL database connection.
func (f *Framework) WithPostgreSQL() *Framework {
	fw, err := f.WithPostgreSQLSafe()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithPostgreSQLSafe sets up the PostgreSQL database connection with error handling.
func (f *Framework) WithPostgreSQLSafe() (*Framework, error) {
	if f.config == nil {
		return f, ErrConfigNotSet
	}

	db, err := NewPostgreSQLFromConfig(f.config)
	if err != nil {
		return f, fmt.Errorf("failed to create PostgreSQL database connection: %w", err)
	}

	f.pgDB = db
	f.AddCleanup(func() error {
		return CloseDB(f.pgDB)
	})

	return f, nil
}

// WithDatabaseFromInstance allows using an existing database connection.
func (f *Framework) WithDatabaseFromInstance(db *sql.DB) *Framework {
	f.db = db
	return f
}

// WithKafkaWriterRoundRobin sets up the Kafka writer.
func (f *Framework) WithKafkaWriterRoundRobin() *Framework {
	fw, err := f.WithKafkaWriterSafeRoundRobin()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithKafkaWriterHash sets up the Kafka writer with Hash balancer.
func (f *Framework) WithKafkaWriterHash() *Framework {
	fw, err := f.WithKafkaWriterSafeHash()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithKafkaWriterSafeHash sets up the Kafka writer with Hash balancer with error handling.
func (f *Framework) WithKafkaWriterSafeHash() (*Framework, error) {
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
func (f *Framework) WithKafkaWriterSafeRoundRobin() (*Framework, error) {
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
func (f *Framework) WithKafkaWriterFromInstance(writer *kafka.Writer) *Framework {
	f.kafkaWriter = writer
	return f
}

// WithProducer sets up the Kafka producer.
func (f *Framework) WithProducer() *Framework {
	fw, err := f.WithProducerSafe()
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithProducerSafe sets up the Kafka producer with error handling.
func (f *Framework) WithProducerSafe() (*Framework, error) {
	if f.kafkaWriter == nil {
		return f, ErrKafkaNotSet
	}

	f.producer = NewKafkaProducer(f.kafkaWriter)
	return f, nil
}

// WithProducerFromInstance allows using an existing producer instance.
func (f *Framework) WithProducerFromInstance(producer *KafkaProducer) *Framework {
	f.producer = producer
	return f
}

// WithWorkerPool sets up the worker pool.
func (f *Framework) WithWorkerPool(numWorkers, queueSize, numShards int) *Framework {
	f.workerPool = NewWorkerPool(numWorkers, queueSize, numShards)
	return f
}

// WithWorkerPoolFromInstance allows using an existing worker pool.
func (f *Framework) WithWorkerPoolFromInstance(pool *ShardedWorkerPool) *Framework {
	f.workerPool = pool
	return f
}

// WithPoller sets up the poller.
func (f *Framework) WithPoller(config PollerConfig, dataFetcher DataFetcher) *Framework {
	fw, err := f.WithPollerSafe(config, dataFetcher)
	if err != nil {
		panic(err.Error())
	}
	return fw
}

// WithPollerSafe sets up the poller with error handling.
func (f *Framework) WithPollerSafe(config PollerConfig, dataFetcher DataFetcher) (*Framework, error) {
	if f.workerPool == nil {
		return f, ErrWorkerPoolNotSet
	}

	f.poller = NewPoller(config, f.workerPool, dataFetcher)
	return f, nil
}

// WithPollerFromInstance allows using an existing poller instance.
func (f *Framework) WithPollerFromInstance(poller *Poller) *Framework {
	f.poller = poller
	return f
}

// WithHttpServer sets up the Fiber-compatible HTTP server.
func (f *Framework) WithHttpServer(setupFn ...func(*HttpServer)) *Framework {
	if f.config == nil {
		panic(ErrConfigNotSet)
	}

	f.httpServer = NewHttpServer()
	if len(setupFn) > 0 {
		setupFn[0](f.httpServer)
	}

	f.AddCleanup(func() error {
		return f.httpServer.Shutdown()
	})

	return f
}

// WithPersistentQueue sets the persistent queue for the framework.
func (f *Framework) WithPersistentQueue(pq PersistentQueue) *Framework {
	f.persistentQueue = pq
	return f
}

// AddCleanup adds a cleanup function to be called during shutdown.
func (f *Framework) AddCleanup(fn func() error) *Framework {
	f.cleanupFuncs = append(f.cleanupFuncs, fn)
	return f
}

// Build finalizes the configuration and returns FrameworkComponents.
func (f *Framework) Build() *FrameworkComponents {
	components, err := f.BuildSafe()
	if err != nil {
		panic(err.Error())
	}
	return components
}

// BuildSafe finalizes the configuration and returns FrameworkComponents.
func (f *Framework) BuildSafe() (*FrameworkComponents, error) {
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

	// Initialize Logger
	logManager := NewLogManager(f.config)
	f.AddCleanup(func() error {
		logManager.Shutdown()
		return nil
	})

	// Inject LogManager into components
	if f.workerPool != nil {
		f.workerPool.WithLogger(logManager)
	}
	if f.poller != nil {
		f.poller.WithLogger(logManager)
	}
	if f.httpServer != nil {
		f.httpServer.WithLogger(logManager)
	}

	if f.workerPool != nil && f.persistentQueue != nil {
		f.workerPool.WithPersistentQueue(f.persistentQueue)
	}

	// Dynamically adjust Kafka BatchSize based on worker pool size
	if f.kafkaWriter != nil && f.workerPool != nil {
		f.kafkaWriter.BatchSize = f.workerPool.NumWorkers()
	}

	return &FrameworkComponents{
		Config:          f.config,
		DB:              f.db,
		PostgreSQL:      f.pgDB,
		KafkaWriter:     f.kafkaWriter,
		Producer:        f.producer,
		WorkerPool:      f.workerPool,
		Poller:          f.poller,
		HttpServer:      f.httpServer,
		Logger:          logManager,
		cleanupFuncs:    f.cleanupFuncs,
		PersistentQueue: f.persistentQueue,
	}, nil
}

// FrameworkComponents holds all initialized framework components.
type FrameworkComponents struct {
	Config          *Config
	DB              *sql.DB
	PostgreSQL      *sql.DB
	KafkaWriter     *kafka.Writer
	Producer        *KafkaProducer
	WorkerPool      *ShardedWorkerPool
	Poller          *Poller
	HttpServer      *HttpServer
	Logger          *LogManager
	PersistentQueue PersistentQueue

	cleanupFuncs []func() error
}

// Start starts all framework components.
func (fc *FrameworkComponents) Start(ctx context.Context, handler JobHandler) error {
	if err := fc.CheckReadiness(ctx); err != nil {
		return fmt.Errorf("framework readiness check failed: %w", err)
	}

	if fc.WorkerPool != nil && fc.PersistentQueue != nil {
		if err := fc.WorkerPool.RestorePending(ctx); err != nil {
			return fmt.Errorf("restore pending jobs: %w", err)
		}
	}

	if fc.WorkerPool != nil {
		fc.WorkerPool.Start(ctx, handler)
	}

	if fc.Poller != nil {
		fc.Poller.Start()
	}

	if fc.HttpServer != nil {
		go func() {
			addr := ":" + fc.Config.Http.Port
			if err := fc.HttpServer.Listen(addr); err != nil && err != http.ErrServerClosed {
				fc.Logger.LogFramework("ERROR", "HttpServer", "Failed to start: "+err.Error())
			}
		}()
	}

	return nil
}

// CheckReadiness performs health checks on all initialized components.
func (fc *FrameworkComponents) CheckReadiness(ctx context.Context) error {
	if fc.PostgreSQL != nil {
		if err := fc.PostgreSQL.PingContext(ctx); err != nil {
			return fmt.Errorf("postgresql not ready: %w", err)
		}
	}

	if fc.DB != nil {
		if err := fc.DB.PingContext(ctx); err != nil {
			return fmt.Errorf("database not ready: %w", err)
		}
	}

	if fc.KafkaWriter != nil {
		if fc.KafkaWriter.Addr == nil || len(fc.KafkaWriter.Addr.String()) == 0 {
			return fmt.Errorf("kafka writer has no brokers configured")
		}
	}

	return nil
}

// Shutdown gracefully shuts down all framework components.
func (fc *FrameworkComponents) Shutdown(ctx context.Context) error {
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
