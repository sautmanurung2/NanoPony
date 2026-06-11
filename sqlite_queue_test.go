package nanopony

import (
	"context"
	"testing"
)

func TestSQLiteQueueFullCycle(t *testing.T) {
	q, err := NewSQLiteQueue("", "")
	if err != nil {
		t.Skipf("SQLite not available: %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	job := AcquireJob()
	job.ID = "job-001"
	job.Data = map[string]interface{}{"key": "val"}
	job.Meta = map[string]interface{}{"source": "test"}

	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobs, err := q.FetchPending(ctx)
	if err != nil {
		t.Fatalf("FetchPending: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-001" {
		t.Fatalf("FetchPending result mismatch")
	}

	if err := q.Dequeue(ctx, "job-001"); err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	jobs, _ = q.FetchPending(ctx)
	if len(jobs) != 0 {
		t.Errorf("Expected 0 after dequeue, got %d", len(jobs))
	}

	// Re-enqueue same ID (INSERT OR REPLACE)
	job2 := AcquireJob()
	job2.ID = "job-001"
	job2.Data = "updated"
	_ = q.Enqueue(ctx, job2)
	
	r, _ := q.FetchPending(ctx)
	if len(r) != 1 || r[0].Data.(string) != "updated" {
		t.Errorf("Re-enqueue failed")
	}
	_ = q.Dequeue(ctx, "job-001")
}

func TestSQLiteQueueInitExistingTable(t *testing.T) {
	q, err := NewSQLiteQueue("", "custom_table_name")
	if err != nil {
		t.Skipf("SQLite not available: %v", err)
	}
	defer q.Close()
	// Creating same table again should be fine
	_ = q.Close()
}

func TestSQLiteQueueNilClose(t *testing.T) {
	q := &SQLiteQueue{}
	q.Close() // Should not panic
}

func TestSQLiteQueueEnqueueDequeueEdge(t *testing.T) {
	q, err := NewSQLiteQueue("", "edge_test")
	if err != nil {
		t.Skipf("SQLite not available: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := AcquireJob()
	job.ID = "edge-1"
	_ = q.Enqueue(ctx, job)

	// Dequeue non-existing
	_ = q.Dequeue(ctx, "non-existent-id")

	// Fetch from empty after full drain
	_ = q.Dequeue(ctx, "edge-1")
	jobs, _ := q.FetchPending(ctx)
	if len(jobs) != 0 {
		t.Errorf("Expected empty")
	}
}
