package nanopony

import (
	"database/sql"
	"fmt"
)

// Repository defines the base interface for all repositories
type Repository interface {
	Close() error
}

// BaseRepository provides common repository functionality
type BaseRepository struct {
	DB *sql.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{
		DB: db,
	}
}

// Close implements Repository interface
func (r *BaseRepository) Close() error {
	return nil
}

// QueryExecutor provides methods for executing queries
type QueryExecutor interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

// TransactionExecutor provides transaction support
type TransactionExecutor interface {
	BeginTx() (*sql.Tx, error)
	WithTransaction(fn func(tx *sql.Tx) error) error
}

// BaseTransactionExecutor provides basic transaction functionality
type BaseTransactionExecutor struct {
	DB *sql.DB
}

// NewTransactionExecutor creates a new transaction executor
func NewTransactionExecutor(db *sql.DB) *BaseTransactionExecutor {
	return &BaseTransactionExecutor{DB: db}
}

// BeginTx starts a new transaction
func (e *BaseTransactionExecutor) BeginTx() (*sql.Tx, error) {
	return e.DB.Begin()
}

// WithTransaction executes a function within a transaction
func (e *BaseTransactionExecutor) WithTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := e.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
