package nanopony

import (
	"context"
	"sync"
)

// Job represents a unit of work to be processed by the worker pool.
// It contains a unique identifier, payload data of type T, and optional metadata.
// Using generics (T any) avoids interface boxing overhead for the Data field.
type Job[T any] struct {
	// ID is a unique identifier for the job
	ID string
	// Data contains the payload of type T to be processed
	Data T
	// Meta contains optional metadata (e.g., source, timestamp)
	Meta map[string]any
}

// jobPools stores a sync.Pool for each job type to allow reuse.
// Since Go doesn't support generic package-level variables in a way that allows
// a single pool for all Job[T], we might need a different strategy if T varies wildly.
// However, for a given application instance, T is usually fixed.
// To keep the API simple and compatible with sync.Pool, we use a registry or 
// expect the user to manage the pool if they use multiple types.
// For NanoPony's primary use case (poller -> worker), T is typically fixed per pool.

// To maintain the AcquireJob/Release pattern while supporting generics, 
// we can use a specialized pool manager or accept that some boxing might 
// happen at the pool level, but NOT at the Data field level during processing.
// A better approach for a generic library is to provide the Pool as part of the WorkerPool.

var jobPools sync.Map // Map of reflect.Type to *sync.Pool

func getJobPool[T any]() *sync.Pool {
	// In a real implementation, we'd use a type key. 
	// For simplicity in this refactor, we'll use a local pool strategy or 
	// allow the Job[T] to be managed by the WorkerPool[T].
	return nil
}

// AcquireJob is now type-specific.
// Note: sync.Pool with generics is tricky because sync.Pool stores 'any'.
func AcquireJob[T any]() *Job[T] {
	// For now, we'll just allocate. A more sophisticated sync.Pool wrapper 
	// for generics can be added later if needed.
	return &Job[T]{
		Meta: make(map[string]any),
	}
}

// Release returns the job to the pool.
func (j *Job[T]) Release() {
	if j == nil {
		return
	}
	// Reset fields.
	j.ID = ""
	var zero T
	j.Data = zero
	j.Meta = make(map[string]any)
	// Putting back into a generic pool is complex due to sync.Pool's any.
}

// JobHandler defines the function signature for processing jobs.
type JobHandler[T any] func(ctx context.Context, job *Job[T]) error
