// poller.go — Periodic data fetching with rate-limited job submission.
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

// DefaultPollerConfig returns default poller configuration.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		Interval:    1 * time.Second,
		BatchSize:   100,
		JobSlotSize: 1,
	}
}

// DataFetcher defines the interface for fetching data during polling.
type DataFetcher[T any] interface {
	// Fetch retrieves a slice of items to be processed as jobs.
	Fetch() ([]T, error)
}

// DataFetcherFunc is a function adapter.
type DataFetcherFunc[T any] func() ([]T, error)

// Fetch implements the DataFetcher interface.
func (f DataFetcherFunc[T]) Fetch() ([]T, error) {
	return f()
}

// Poller manages periodic data fetching.
type Poller[T any] struct {
	config      PollerConfig
	jobSlots    chan struct{}
	workerPool  *ShardedWorkerPool[T]
	dataFetcher DataFetcher[T]
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
	jobCounter  int64
	sessionID   string
}

// NewPoller creates a new Poller instance.
func NewPoller[T any](config PollerConfig, workerPool *ShardedWorkerPool[T], dataFetcher DataFetcher[T]) *Poller[T] {
	ctx, cancel := context.WithCancel(context.Background())
	sessionID := time.Now().Format("20060102150405")

	poller := &Poller[T]{
		config:      config,
		jobSlots:    make(chan struct{}, config.JobSlotSize),
		workerPool:  workerPool,
		dataFetcher: dataFetcher,
		ctx:         ctx,
		cancel:      cancel,
		sessionID:   sessionID,
	}

	for i := 0; i < config.JobSlotSize; i++ {
		poller.jobSlots <- struct{}{}
	}

	return poller
}

// Start initiates the polling loop.
func (p *Poller[T]) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.wg.Add(1)

	go p.pollLoop()
}

func (p *Poller[T]) pollLoop() {
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

func (p *Poller[T]) pollOnce() {
	select {
	case <-p.jobSlots:
	default:
		return
	}

	defer p.releaseSlot()

	data, err := p.dataFetcher.Fetch()
	if err != nil {
		LogFramework("ERROR", "Poller", fmt.Sprintf("failed to fetch data: %v", err))
		return
	}

	if len(data) == 0 {
		return
	}

	if p.config.BatchSize > 0 && len(data) > p.config.BatchSize {
		LogFramework("WARNING", "Poller", fmt.Sprintf("limiting batch size from %d to %d", len(data), p.config.BatchSize))
		data = data[:p.config.BatchSize]
	}

	for _, item := range data {
		jobID := atomic.AddInt64(&p.jobCounter, 1)
		job := AcquireJob[T]()

		var sb strings.Builder
		sb.WriteString("poll-")
		sb.WriteString(p.sessionID)
		sb.WriteString("-")
		sb.WriteString(strconv.FormatInt(jobID, 10))
		job.ID = sb.String()

		job.Data = item
		job.Meta["source"] = "poller"
		job.Meta["timestamp"] = time.Now().Unix()

		if err := p.workerPool.SubmitBlocking(p.ctx, job); err != nil {
			if p.ctx.Err() == nil {
				LogFramework("ERROR", "Poller", fmt.Sprintf("failed to submit job: %v", err))
			}
			job.Release()
		}
	}
}

func (p *Poller[T]) releaseSlot() {
	select {
	case p.jobSlots <- struct{}{}:
	default:
	}
}

func (p *Poller[T]) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.cancel()
	p.wg.Wait()
	p.running = false
}

func (p *Poller[T]) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
