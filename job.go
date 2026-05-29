package nanopony

import (
	"context"
	"sync"
)

// Job represents a unit of work to be processed by the worker pool.
// It contains a unique ID, payload data, and optional metadata.
type Job struct {
	// ID is a unique identifier for the job
	ID string
	// Data contains the payload to be processed
	Data any
	// Meta contains optional metadata (e.g., source, timestamp)
	Meta map[string]any
}

var jobPool = sync.Pool{
	New: func() any {
		return &Job{
			Meta: make(map[string]any),
		}
	},
}

// AcquireJob retrieves a Job from the pool.
// Always call job.Release() when finished with the job to return it to the pool.
func AcquireJob() *Job {
	return jobPool.Get().(*Job)
}

// Release returns the job to the pool after resetting its fields.
// Do not use the job after calling Release.
func (j *Job) Release() {
	if j == nil {
		return
	}
	j.ID = ""
	j.Data = nil
	// Recreating the map is generally faster than clearing it key-by-key for large maps
	// and helps manage memory better.
	j.Meta = make(map[string]any)
	jobPool.Put(j)
}

// JobHandler defines the function signature for processing jobs.
// It receives a context and a job pointer, and returns an error if processing fails.
type JobHandler func(ctx context.Context, job *Job) error
