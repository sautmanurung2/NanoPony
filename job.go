package nanopony

import (
	"context"
)

// Job represents a unit of work to be processed by the worker pool.
// It contains a unique ID, payload data, and optional metadata.
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

// JobHandler defines the function signature for processing jobs.
// It receives a context and a job, and returns an error if processing fails.
//
// Example:
//
//	handler := func(ctx context.Context, job Job) error {
//	    log.Printf("Processing job %s: %+v", job.ID, job.Data)
//	    return nil
//	}
type JobHandler func(ctx context.Context, job Job) error
