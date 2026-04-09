package nanopony

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Job represents a unit of work to be processed by the worker pool.
//
// Example:
//
//	job := Job{
//	    ID:   "job-1",
//	    Data: map[string]interface{}{"key": "value"},
//	    Meta: map[string]interface{}{"source": "poller"},
//	}
type Job struct {
	// ID is a unique identifier for the job
	ID string
	// Data contains the payload to be processed
	Data any
	// Meta contains optional metadata (e.g., source, timestamp)
	Meta map[string]any
}

// JobHandler defines the handler for processing jobs.
// It receives a context and a job, and returns an error if processing fails.
//
// Example:
//
//	handler := func(ctx context.Context, job Job) error {
//	    log.Printf("Processing job %s: %+v", job.ID, job.Data)
//	    // Process the job...
//	    return nil
//	}
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

// WorkerPool manages a pool of workers for concurrent job processing.
// It uses a bounded queue to prevent memory issues under high load.
//
// Example:
//
//	pool := NewWorkerPool(5, 100) // 5 workers, queue size 100
//	pool.Start(ctx, handler)
//	defer pool.Stop()
//
//	// Submit jobs
//	pool.Submit(ctx, Job{Data: "task1"})
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

// NewWorkerPool creates a new worker pool with the specified number of workers and queue size.
//
// Parameters:
//   - numWorkers: Number of concurrent goroutines to process jobs
//   - queueSize: Maximum number of jobs that can be queued
//
// Example:
//
//	pool := NewWorkerPool(5, 100)
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

// Start starts the worker pool with the given handler.
// The pool will spawn numWorkers goroutines to process jobs concurrently.
//
// If the pool is already running, this method is a no-op.
func (wp *WorkerPool) Start(ctx context.Context, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return
	}

	wp.handler = handler
	wp.running = true

	// Spawn worker goroutines
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// worker is a goroutine that processes jobs from the job channel.
// It listens for context cancellation or job arrival.
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, shutdown worker
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				// Channel closed, shutdown worker
				return
			}

			// Process the job and handle errors
			if err := wp.handler(ctx, job); err != nil {
				// Try to send error to error channel
				select {
				case wp.errChan <- fmt.Errorf("worker %d: job %s failed: %w", id, job.ID, err):
					// Error sent successfully
				default:
					// Error channel full, log warning instead of silently discarding
					log.Printf("[WARNING] Error channel full, discarding error: %v", err)
				}
			}
		}
	}
}

// Submit submits a job to the worker pool for processing.
// This method is non-blocking and will return ErrQueueFull if the queue is full.
//
// Example:
//
//	err := pool.Submit(ctx, Job{
//	    ID:   "job-1",
//	    Data: map[string]interface{}{"key": "value"},
//	})
//	if err == nanopony.ErrQueueFull {
//	    log.Println("Queue is full, please retry later")
//	}
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

// Stop stops the worker pool gracefully.
// It cancels the context, closes the job channel, and waits for all workers to finish.
// If the pool is not running, this method is a no-op.
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return
	}

	// Cancel context to signal workers to stop
	wp.cancel()
	// Close job channel to prevent new submissions
	close(wp.jobChan)
	// Wait for all workers to finish
	wp.wg.Wait()
	// Close error channel
	close(wp.errChan)
	wp.running = false
}

// Errors returns the error channel for monitoring job processing errors.
// Consumers should range over this channel to receive errors.
//
// Example:
//
//	go func() {
//	    for err := range pool.Errors() {
//	        log.Printf("Job error: %v", err)
//	    }
//	}()
func (wp *WorkerPool) Errors() <-chan error {
	return wp.errChan
}

// IsRunning returns whether the worker pool is currently running
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.running
}

// PollerConfig holds configuration for polling
type PollerConfig struct {
	// Interval is the time between poll attempts
	Interval time.Duration
	// MaxRetries is the maximum number of retries on failure (currently unused)
	MaxRetries int
	// RetryDelay is the delay between retries (currently unused)
	RetryDelay time.Duration
	// BatchSize is the maximum number of items to process per batch (currently unused)
	BatchSize int
	// JobSlotSize controls how many concurrent poll operations are allowed
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

// Poller manages periodic data fetching and job submission to the worker pool.
// It fetches data at configured intervals and submits each data item as a job.
//
// Example:
//
//	dataFetcher := DataFetcherFunc(func() ([]any, error) {
//	    return fetchDataFromDatabase()
//	})
//
//	poller := NewPoller(config, workerPool, dataFetcher)
//	poller.Start()
//	defer poller.Stop()
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

// DataFetcher defines the interface for fetching data during polling.
// Implement this interface to provide custom data sources.
//
// Example using function adapter:
//
//	fetcher := DataFetcherFunc(func() ([]any, error) {
//	    rows, err := db.Query("SELECT * FROM table")
//	    // ... process rows
//	    return data, nil
//	})
type DataFetcher interface {
	Fetch() ([]any, error)
}

// DataFetcherFunc is a function adapter that allows regular functions
// to implement the DataFetcher interface.
type DataFetcherFunc func() ([]any, error)

// Fetch implements DataFetcher interface
func (f DataFetcherFunc) Fetch() ([]any, error) {
	return f()
}

// NewPoller creates a new poller with the given configuration, worker pool, and data fetcher.
//
// Parameters:
//   - config: PollerConfig with interval, batch size, etc.
//   - workerPool: WorkerPool to submit jobs to
//   - dataFetcher: DataFetcher to fetch data from
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

	// Pre-fill job slots (acts as semaphore for rate limiting)
	for i := 0; i < config.JobSlotSize; i++ {
		poller.jobSlots <- struct{}{}
	}

	return poller
}

// Start starts the polling process.
// If the poller is already running, this method is a no-op.
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

// poll is the main polling loop that runs until the poller is stopped
func (p *Poller) poll() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// Context cancelled, stop polling
			return
		case <-ticker.C:
			// Tick received, perform one poll iteration
			p.pollOnce()
		}
	}
}

// pollOnce performs a single poll iteration
func (p *Poller) pollOnce() {
	// Try to acquire a job slot (rate limiting)
	select {
	case <-p.jobSlots:
		// Slot acquired, proceed with polling
	default:
		// No slot available, skip this iteration
		return
	}

	// Fetch data from source
	data, err := p.dataFetcher.Fetch()
	if err != nil {
		log.Printf("[ERROR] Poller: failed to fetch data: %v", err)
		p.releaseSlot()
		return
	}

	// Check if data is empty
	if len(data) == 0 {
		p.releaseSlot()
		return
	}

	// Submit each data item as a job
	for i, item := range data {
		job := Job{
			ID:   fmt.Sprintf("poll-%d", i),
			Data: item,
			Meta: map[string]any{
				"source":    "poller",
				"timestamp": time.Now().Unix(),
			},
		}

		if err := p.workerPool.Submit(p.ctx, job); err != nil {
			// Log job submission failures instead of silently skipping
			log.Printf("[ERROR] Poller: failed to submit job: %v", err)
		}
	}

	// Release the slot after all jobs are submitted
	p.releaseSlot()
}

// releaseSlot releases a job slot back to the pool
func (p *Poller) releaseSlot() {
	select {
	case p.jobSlots <- struct{}{}:
	default:
	}
}

// Stop stops the poller gracefully.
// It cancels the context and waits for the polling goroutine to finish.
// If the poller is not running, this method is a no-op.
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

// IsRunning returns whether the poller is currently running
func (p *Poller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
