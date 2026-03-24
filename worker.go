package nanopony

import (
	"context"
	"sync"
	"time"
)

// Job represents a unit of work to be processed
type Job struct {
	ID   string
	Data interface{}
	Meta map[string]interface{}
}

// JobHandler defines the handler for processing jobs
type JobHandler func(ctx context.Context, job Job) error

// WorkerPoolConfig holds configuration for worker pool
type WorkerPoolConfig struct {
	NumWorkers int
	QueueSize  int
}

// DefaultWorkerPoolConfig returns default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		NumWorkers: 5,
		QueueSize:  100,
	}
}

// WorkerPool manages a pool of workers for processing jobs
type WorkerPool struct {
	numWorkers int
	jobChan    chan Job
	errChan    chan error
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	handler    JobHandler
	mu         sync.RWMutex
	running    bool
}

// NewWorkerPool creates a new worker pool with the specified number of workers and queue size
func NewWorkerPool(numWorkers, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		numWorkers: numWorkers,
		jobChan:    make(chan Job, queueSize),
		errChan:    make(chan error, queueSize),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker pool with the given handler
func (wp *WorkerPool) Start(ctx context.Context, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}

	wp.handler = handler
	wp.running = true

	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// worker is a goroutine that processes jobs
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				return
			}

			if err := wp.handler(ctx, job); err != nil {
				select {
				case wp.errChan <- err:
				default:
					// Error channel full, discard error
				}
			}
		}
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case wp.jobChan <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return
	}

	wp.cancel()
	close(wp.jobChan)
	wp.wg.Wait()
	close(wp.errChan)
	wp.running = false
}

// Errors returns the error channel
func (wp *WorkerPool) Errors() <-chan error {
	return wp.errChan
}

// IsRunning returns whether the worker pool is running
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}

// PollerConfig holds configuration for polling
type PollerConfig struct {
	Interval    time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	BatchSize   int
	JobSlotSize int
}

// DefaultPollerConfig returns default poller configuration
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		Interval:    1 * time.Second,
		MaxRetries:  3,
		RetryDelay:  100 * time.Millisecond,
		BatchSize:   100,
		JobSlotSize: 1,
	}
}

// Poller manages polling for data and submitting jobs to worker pool
type Poller struct {
	config      PollerConfig
	jobSlots    chan struct{}
	workerPool  *WorkerPool
	dataFetcher DataFetcher
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
}

// DataFetcher defines the interface for fetching data
type DataFetcher interface {
	Fetch() ([]interface{}, error)
}

// DataFetcherFunc is a function adapter for DataFetcher
type DataFetcherFunc func() ([]interface{}, error)

// Fetch implements DataFetcher
func (f DataFetcherFunc) Fetch() ([]interface{}, error) {
	return f()
}

// NewPoller creates a new poller
func NewPoller(config PollerConfig, workerPool *WorkerPool, dataFetcher DataFetcher) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	poller := &Poller{
		config:      config,
		jobSlots:    make(chan struct{}, config.JobSlotSize),
		workerPool:  workerPool,
		dataFetcher: dataFetcher,
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < config.JobSlotSize; i++ {
		poller.jobSlots <- struct{}{}
	}

	return poller
}

// Start starts the polling process
func (p *Poller) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.wg.Add(1)

	go p.poll()
}

// poll is the main polling loop
func (p *Poller) poll() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce()
		}
	}
}

// pollOnce performs a single poll iteration
func (p *Poller) pollOnce() {
	select {
	case <-p.jobSlots:
	default:
		return
	}

	data, err := p.dataFetcher.Fetch()
	if err != nil {
		p.releaseSlot()
		return
	}

	if len(data) == 0 {
		p.releaseSlot()
		return
	}

	for _, item := range data {
		job := Job{
			Data: item,
			Meta: make(map[string]interface{}),
		}

		if err := p.workerPool.Submit(p.ctx, job); err != nil {
			continue
		}
	}
}

// releaseSlot releases a job slot
func (p *Poller) releaseSlot() {
	select {
	case p.jobSlots <- struct{}{}:
	default:
	}
}

// Stop stops the poller
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.cancel()
	p.wg.Wait()
	p.running = false
}

// IsRunning returns whether the poller is running
func (p *Poller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
