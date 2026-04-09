package nanopony

import (
	"database/sql"
	"fmt"
)

// Repository defines the base interface for all repositories.
// All repositories should implement this interface to ensure proper cleanup.
//
// Example implementation:
//
//	type UserRepository struct {
//	    *nanopony.BaseRepository
//	}
//
//	func (r *UserRepository) Close() error {
//	    // Custom cleanup if needed
//	    return nil
//	}
type Repository interface {
	// Close releases repository resources
	Close() error
}

// BaseRepository provides common repository functionality.
// Embed this in your custom repositories to get basic functionality.
//
// Example:
//
//	type UserRepository struct {
//	    *nanopony.BaseRepository
//	}
//
//	func (r *UserRepository) GetUser(id int) (*User, error) {
//	    row := r.DB.QueryRow("SELECT * FROM users WHERE id = :id", sql.Named("id", id))
//	    // Process row...
//	}
type BaseRepository struct {
	// DB is the database connection
	DB *sql.DB
}

// NewBaseRepository creates a new base repository with the given database connection
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{
		DB: db,
	}
}

// Close implements Repository interface.
// Returns nil as BaseRepository doesn't own the DB connection.
func (r *BaseRepository) Close() error {
	return nil
}

// QueryExecutor provides methods for executing queries.
// This interface abstracts database query operations.
type QueryExecutor interface {
	// Query executes a query that returns multiple rows
	Query(query string, args ...any) (*sql.Rows, error)
	// QueryRow executes a query that returns a single row
	QueryRow(query string, args ...any) *sql.Row
	// Exec executes a query that doesn't return rows (INSERT, UPDATE, DELETE)
	Exec(query string, args ...any) (sql.Result, error)
	// Prepare creates a prepared statement for repeated execution
	Prepare(query string) (*sql.Stmt, error)
}

// TransactionExecutor provides transaction support.
// Use this to execute multiple database operations atomically.
type TransactionExecutor interface {
	// BeginTx starts a new transaction
	BeginTx() (*sql.Tx, error)
	// WithTransaction executes a function within a transaction.
	// Automatically commits on success or rollbacks on error.
	WithTransaction(fn func(tx *sql.Tx) error) error
}

// BaseTransactionExecutor provides basic transaction functionality.
// It handles the transaction lifecycle: begin, commit/rollback.
//
// Example:
//
//	executor := NewTransactionExecutor(db)
//	err := executor.WithTransaction(func(tx *sql.Tx) error {
//	    _, err := tx.Exec("INSERT INTO users (name) VALUES (?)", "John")
//	    if err != nil {
//	        return err
//	    }
//	    _, err = tx.Exec("INSERT INTO profiles (user_id) VALUES (LAST_INSERT_ID())")
//	    return err
//	})
type BaseTransactionExecutor struct {
	// DB is the database connection
	DB *sql.DB
}

// NewTransactionExecutor creates a new transaction executor with the given database connection
func NewTransactionExecutor(db *sql.DB) *BaseTransactionExecutor {
	return &BaseTransactionExecutor{DB: db}
}

// BeginTx starts a new transaction
func (e *BaseTransactionExecutor) BeginTx() (*sql.Tx, error) {
	return e.DB.Begin()
}

// WithTransaction executes a function within a transaction.
// Transaction lifecycle:
// 1. Begin transaction
// 2. Execute the function
// 3. If function succeeds: Commit transaction
// 4. If function fails: Rollback transaction and return error
//
// The function receives a transaction object that should be used for all database operations.
// If the function returns an error, the transaction is automatically rolled back.
func (e *BaseTransactionExecutor) WithTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := e.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer rollback - if commit succeeds, this is a no-op
	// If function fails, this ensures transaction is rolled back
	defer func() { _ = tx.Rollback() }()

	// Execute the function within the transaction
	if err := fn(tx); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
