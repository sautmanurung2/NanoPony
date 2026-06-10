// poller.go — Periodic data fetching with rate-limited job submission.
//
// Architecture:
//
//	ticker (every Interval)
//	   ↓
//	pollOnce()
//	   ├── acquire jobSlot (semaphore — limits concurrent polls)
//	   ├── dataFetcher.Fetch()  → []any
//	   ├── for each item → workerPool.SubmitBlocking(job)
//	   └── release jobSlot
//
// The jobSlots channel acts as a counting semaphore:
//   - Capacity = JobSlotSize (default 1)
//   - If no slot available, the tick is skipped (prevents pile-up)
//   - This is NOT the same as WorkerPool queue size
package nanopony

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PollerConfig holds configuration for the polling process.
type PollerConfig struct {
	// Interval is the time between poll attempts
	Interval time.Duration
	// BatchSize is the maximum number of items to process per batch iteration
	BatchSize int
	// JobSlotSize controls how many concurrent poll operations are allowed to fetch data
	JobSlotSize int
}

// DefaultPollerConfig returns default poller configuration with sensible defaults.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		Interval:    1 * time.Second,
		BatchSize:   100,
		JobSlotSize: 1,
	}
}

// DataFetcher defines the interface for fetching data during polling.
// Implement this interface to provide custom data sources (e.g., Database, API).
type DataFetcher interface {
	// Fetch retrieves a slice of items to be processed as jobs.
	// It accepts a context for cancellation support during long-running fetches.
	Fetch(ctx context.Context) ([]any, error)
}

// DataFetcherFunc is a function adapter that allows regular functions
// to implement the DataFetcher interface.
type DataFetcherFunc func(ctx context.Context) ([]any, error)

// Fetch implements the DataFetcher interface by calling the adapter function.
func (f DataFetcherFunc) Fetch(ctx context.Context) ([]any, error) {
	return f(ctx)
}

// Poller manages periodic data fetching and submission of jobs to a WorkerPool.
// It uses a semaphore-based rate limiting system (jobSlots) to control concurrency.
type Poller struct {
	config      PollerConfig
	jobSlots    chan struct{}
	workerPool  *ShardedWorkerPool
	dataFetcher DataFetcher
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
	jobCounter  int64  // Atomic counter for unique job IDs within a session
	sessionID   string // Unique session ID for job ID prefix (e.g., timestamp)
}

// NewPoller creates a new Poller instance.
//
// Parameters:
//   - config: Configuration settings for timing and batching
//   - workerPool: The pool where fetched items will be submitted as jobs
//   - dataFetcher: The source of data items
func NewPoller(config PollerConfig, workerPool *ShardedWorkerPool, dataFetcher DataFetcher) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	// Use timestamp for session ID to ensure unique job IDs across restarts
	sessionID := time.Now().Format("20060102150405")

	poller := &Poller{
		config:      config,
		jobSlots:    make(chan struct{}, config.JobSlotSize),
		workerPool:  workerPool,
		dataFetcher: dataFetcher,
		ctx:         ctx,
		cancel:      cancel,
		sessionID:   sessionID,
	}

	// Pre-fill job slots (acts as semaphore for rate limiting)
	for i := 0; i < config.JobSlotSize; i++ {
		poller.jobSlots <- struct{}{}
	}

	return poller
}

// Start initiates the polling loop in a separate goroutine.
// If the poller is already running, this method does nothing.
func (p *Poller) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.wg.Add(1)

	go p.pollLoop()
}

// pollLoop is the main ticker loop for the poller.
func (p *Poller) pollLoop() {
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

// pollOnce performs a single polling iteration:
// 1. Acquire a slot
// 2. Fetch data
// 3. Submit jobs to WorkerPool
// 4. Release slot
func (p *Poller) pollOnce() {
	// Try to acquire a job slot (rate limiting)
	select {
	case <-p.jobSlots:
		// Slot acquired
	default:
		// No slot available, skipping this tick
		return
	}

	// Always release slot when done
	defer p.releaseSlot()

	data, err := p.dataFetcher.Fetch(p.ctx)
	if err != nil {
		LogFramework("ERROR", "Poller", fmt.Sprintf("failed to fetch data: %v", err))
		return
	}

	if len(data) == 0 {
		return
	}

	// Limit batch size if configured
	if p.config.BatchSize > 0 && len(data) > p.config.BatchSize {
		LogFramework("WARNING", "Poller", fmt.Sprintf("limiting batch size from %d to %d", len(data), p.config.BatchSize))
		data = data[:p.config.BatchSize]
	}

	for _, item := range data {
		jobID := atomic.AddInt64(&p.jobCounter, 1)
		job := AcquireJob()

		// Use strings.Builder with strconv.AppendInt for zero-allocation ID construction
		var sb strings.Builder
		sb.Grow(len("poll-") + len(p.sessionID) + 1 + 20) // 20 is max digits for int64
		sb.WriteString("poll-")
		sb.WriteString(p.sessionID)
		sb.WriteByte('-')

		// Directly append integer to builder buffer to avoid intermediate string allocation
		var buf [20]byte
		b := strconv.AppendInt(buf[:0], jobID, 10)
		sb.Write(b)

		job.ID = sb.String()

		job.Data = item
		job.Meta["source"] = "poller"
		job.Meta["timestamp"] = time.Now().Unix()

		// Use SubmitBlocking to provide backpressure
		if err := p.workerPool.SubmitBlocking(p.ctx, job); err != nil {
			if p.ctx.Err() == nil {
				LogFramework("ERROR", "Poller", fmt.Sprintf("failed to submit job: %v", err))
			}
			// If submission fails, we should release the job as it won't be processed
			job.Release()
		}
	}
}

// releaseSlot returns a token to the jobSlots channel.
func (p *Poller) releaseSlot() {
	select {
	case p.jobSlots <- struct{}{}:
	default:
	}
}

// Stop stops the poller gracefully and waits for the loop to exit.
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

// IsRunning returns the current status of the poller.
func (p *Poller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
