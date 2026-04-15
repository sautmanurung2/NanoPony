package nanopony

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

// oracleDB holds the global database connection.
// This is set by NewOracleConnection and can be accessed via GetOracleDB().
// Protected by dbMutex for thread safety.
var oracleDB *sql.DB
var dbMutex sync.RWMutex

// DatabaseConfig holds Oracle database connection configuration.
// Use DefaultDatabaseConfig() to get sensible defaults.
//
// Example:
//
//	config := DefaultDatabaseConfig()
//	config.Host = "localhost"
//	config.Port = "1521"
//	db, err := NewOracleConnection(config)
type DatabaseConfig struct {
	Host            string
	Port            string
	Database        string
	Username        string
	Password        string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnIdleTime    time.Duration
	ConnMaxLifetime time.Duration
}

// DefaultDatabaseConfig returns default database configuration with sensible pool settings.
// Default pool settings: MaxIdleConns=10, MaxOpenConns=100,
// ConnIdleTime=5min, ConnMaxLifetime=60min
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnIdleTime:    5 * time.Minute,
		ConnMaxLifetime: 60 * time.Minute,
	}
}

// NewOracleConnection creates a new Oracle database connection with connection pooling.
//
// The function:
// 1. Builds the Oracle connection URL
// 2. Opens the database connection
// 3. Configures connection pool settings
// 4. Pings the database to verify the connection
// 5. Sets the global oracleDB variable (for backward compatibility)
//
// Example:
//
//	config := DatabaseConfig{
//	    Host:     "localhost",
//	    Port:     "1521",
//	    Database: "ORCL",
//	    Username: "user",
//	    Password: "secret",
//	}
//	db, err := NewOracleConnection(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func NewOracleConnection(config DatabaseConfig) (*sql.DB, error) {
	port, err := parsePort(config.Port)
	if err != nil {
		log.Printf("[WARNING] Invalid port '%s', using default 1521: %v", config.Port, err)
		port = 1521
	}

	connStr := go_ora.BuildUrl(
		config.Host,
		port,
		config.Database,
		config.Username,
		config.Password,
		nil,
	)

	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open oracle connection: %w", err)
	}

	// Use user-provided pool settings if set, otherwise fall back to defaults
	defaults := DefaultDatabaseConfig()
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(defaults.MaxIdleConns)
	}
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(defaults.MaxOpenConns)
	}
	if config.ConnIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnIdleTime)
	} else {
		db.SetConnMaxIdleTime(defaults.ConnIdleTime)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(defaults.ConnMaxLifetime)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping oracle database: %w", err)
	}

	// Set global variable for backward compatibility (thread-safe)
	// Note: For new code, prefer using the returned *sql.DB directly
	dbMutex.Lock()
	oracleDB = db
	dbMutex.Unlock()

	return db, nil
}

// parsePort converts string port to int, returning error if invalid
func parsePort(portStr string) (int, error) {
	if portStr == "" {
		return 1521, nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number '%s': %w", portStr, err)
	}

	return port, nil
}

// NewOracleFromConfig creates Oracle connection from the application Config.
// This is a convenience wrapper that extracts database config from the main Config.
//
// Example:
//
//	config := nanopony.NewConfig()
//	db, err := nanopony.NewOracleFromConfig(config)
func NewOracleFromConfig(conf *Config) (*sql.DB, error) {
	dbConfig := DatabaseConfig{
		Host:     conf.Oracle.Host,
		Port:     conf.Oracle.Port,
		Database: conf.Oracle.DatabaseName,
		Username: conf.Oracle.Username,
		Password: conf.Oracle.Password,
	}
	return NewOracleConnection(dbConfig)
}

// GetOracleDB returns the global database connection.
// Returns nil if no connection has been established.
// Note: For new code, prefer passing *sql.DB explicitly.
func GetOracleDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return oracleDB
}

// CloseDB safely closes a database connection.
// Returns nil if db is nil (safe to call with nil).
func CloseDB(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// InterpolateQueryForLoggingOnly replaces named parameters in SQL query with actual values.
//
// ⚠️  WARNING: FOR LOGGING/DEBUGGING ONLY. DO NOT use this for executing queries.
// Interpolated queries are NOT safe against SQL injection and should NEVER be executed.
// Use parameterized queries (with sql.Named) for actual database operations.
//
// Example:
//
//	query := "SELECT * FROM users WHERE id = :id AND name = :name"
//	interpolated := InterpolateQueryForLoggingOnly(query,
//	    sql.Named("id", 123),
//	    sql.Named("name", "John"),
//	)
//	// Result: "SELECT * FROM users WHERE id = 123 AND name = 'John'"
func InterpolateQueryForLoggingOnly(query string, args ...any) string {
	return InterpolateQuery(query, args...)
}

// InterpolateQuery replaces named parameters in SQL query with actual values.
// This is useful for debugging and logging queries.
//
// ⚠️  WARNING: This is for LOGGING ONLY, not for executing queries.
// Never use interpolated queries for actual database execution as they are not SQL-injection safe.
//
// Example:
//
//	query := "SELECT * FROM users WHERE id = :id AND name = :name"
//	interpolated := InterpolateQuery(query,
//	    sql.Named("id", 123),
//	    sql.Named("name", "John"),
//	)
//	// Result: "SELECT * FROM users WHERE id = 123 AND name = 'John'"
func InterpolateQuery(query string, args ...any) string {
	for _, arg := range args {
		namedArg, ok := arg.(sql.NamedArg)
		if !ok {
			continue
		}

		value := formatValue(namedArg.Value)
		query = strings.ReplaceAll(query, ":"+namedArg.Name, value)
	}
	return query
}

// formatValue formats a value for SQL query interpolation (for logging purposes).
// It handles strings, numbers, nil, and other types appropriately.
func formatValue(v any) string {
	switch v := v.(type) {
	case string:
		// Escape single quotes by doubling them
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// LogInterpolatedQuery logs the interpolated SQL query with values.
// This is useful for debugging database queries.
//
// ⚠️  WARNING: FOR LOGGING ONLY. The interpolated query is NEVER executed.
// Consider using InterpolateQueryForLoggingOnly() for explicit intent.
//
// Example:
//
//	LogInterpolatedQuery(
//	    "SELECT * FROM users WHERE id = :id",
//	    sql.Named("id", 123),
//	)
//	// Output: [SQL Query] SELECT * FROM users WHERE id = 123
func LogInterpolatedQuery(query string, args ...any) {
	interpolated := InterpolateQuery(query, args...)
	log.Printf("[SQL Query] %s", interpolated)
}
