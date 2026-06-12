package nanopony

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PersistentQueue defines the interface for a persistent job store.
type PersistentQueue interface {
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context, jobID string) error
	FetchPending(ctx context.Context) ([]*Job, error)
	Close() error
}

// SQLiteQueue implements PersistentQueue using SQLite.
type SQLiteQueue struct {
	db        *sql.DB
	tableName string
	mu        sync.Mutex
}

// NewSQLiteQueue creates a new SQLite-backed persistent queue.
func NewSQLiteQueue(dbPath string, tableName string) (*SQLiteQueue, error) {
	if dbPath == "" {
		dbPath = "/app/src/data/jobs.db"
	}
	if tableName == "" {
		tableName = "pending_jobs"
	}
	// Auto-create directory for the database file if it does not exist
	// This allows Kubernetes PVC mounts to work without extra init containers
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create sqlite directory %s: %w", dir, err)
		}
	}
	// Using a generic driver name, assuming user registers a driver like "sqlite3" or "sqlite"
	// For better compatibility, we'll try "sqlite" first then "sqlite3"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite: %w", err)
		}
	}

	q := &SQLiteQueue{
		db:        db,
		tableName: tableName,
	}

	if err := q.init(); err != nil {
		db.Close()
		return nil, err
	}

	return q, nil
}

func (q *SQLiteQueue) init() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			data BLOB,
			meta BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, q.tableName)

	_, err := q.db.Exec(query)
	return err
}

func (q *SQLiteQueue) Enqueue(ctx context.Context, job *Job) error {
	dataBytes, err := json.Marshal(job.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	metaBytes, err := json.Marshal(job.Meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (id, data, meta) VALUES (?, ?, ?)", q.tableName)
	_, err = q.db.ExecContext(ctx, query, job.ID, dataBytes, metaBytes)
	return err
}

func (q *SQLiteQueue) Dequeue(ctx context.Context, jobID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", q.tableName)
	_, err := q.db.ExecContext(ctx, query, jobID)
	return err
}

func (q *SQLiteQueue) FetchPending(ctx context.Context) ([]*Job, error) {
	query := fmt.Sprintf("SELECT id, data, meta FROM %s ORDER BY created_at ASC", q.tableName)
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var id string
		var dataBytes, metaBytes []byte
		if err := rows.Scan(&id, &dataBytes, &metaBytes); err != nil {
			return nil, err
		}

		job := AcquireJob()
		job.ID = id

		var data any
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			job.Release()
			return nil, err
		}
		job.Data = data

		var meta map[string]any
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			job.Release()
			return nil, err
		}
		job.Meta = meta

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (q *SQLiteQueue) Close() error {
	if q.db != nil {
		return q.db.Close()
	}
	return nil
}
