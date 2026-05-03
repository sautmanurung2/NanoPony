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
//
// Deprecated: This global singleton exists for backward compatibility.
// New code should use Framework.WithDatabase() and access db via FrameworkComponents.DB.
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
// Default pool settings: MaxIdleConns=2, MaxOpenConns=20,
// ConnIdleTime=5min, ConnMaxLifetime=60min
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxIdleConns:    2,
		MaxOpenConns:    20,
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

	// Apply pool settings with sensible defaults
	defaults := DefaultDatabaseConfig()
	db.SetMaxIdleConns(GetOrDefault(config.MaxIdleConns, defaults.MaxIdleConns))
	db.SetMaxOpenConns(GetOrDefault(config.MaxOpenConns, defaults.MaxOpenConns))
	db.SetConnMaxIdleTime(GetOrDefault(config.ConnIdleTime, defaults.ConnIdleTime))
	db.SetConnMaxLifetime(GetOrDefault(config.ConnMaxLifetime, defaults.ConnMaxLifetime))

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
	oracle := conf.EnsureOracle()
	dbConfig := DatabaseConfig{
		Host:     oracle.Host,
		Port:     oracle.Port,
		Database: oracle.DatabaseName,
		Username: oracle.Username,
		Password: oracle.Password,
	}
	return NewOracleConnection(dbConfig)
}

// GetOracleDB returns the global database connection.
// Returns nil if no connection has been established.
//
// Deprecated: For new code, use Framework.WithDatabase() and access db
// via FrameworkComponents.DB instead of this global accessor.
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
// Consider using InterpolateQuery() for explicit intent.
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
