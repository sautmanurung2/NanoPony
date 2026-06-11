package nanopony

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	go_ora "github.com/sijms/go-ora/v2"
)

// DatabaseConfig holds database connection configuration for Oracle and PostgreSQL.
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
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxIdleConns:    2,
		MaxOpenConns:    20,
		ConnIdleTime:    5 * time.Minute,
		ConnMaxLifetime: 60 * time.Minute,
	}
}

// NewOracleConnection creates a new Oracle database connection.
func NewOracleConnection(config DatabaseConfig) (*sql.DB, error) {
	port, err := parsePort(config.Port)
	if err != nil {
		log.Printf("[WARNING] Invalid Oracle port '%s', using 1521: %v", config.Port, err)
		port = 1521
	}

	connStr := go_ora.BuildUrl(config.Host, port, config.Database, config.Username, config.Password, nil)
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open oracle connection: %w", err)
	}

	applyPoolSettings(db, config)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping oracle database: %w", err)
	}

	return db, nil
}

// NewPostgreSQLConnection creates a new PostgreSQL database connection.
func NewPostgreSQLConnection(config DatabaseConfig) (*sql.DB, error) {
	port, err := parsePort(config.Port)
	if err != nil {
		log.Printf("[WARNING] Invalid PostgreSQL port '%s', using 5432: %v", config.Port, err)
		port = 5432
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host, port, config.Username, config.Password, config.Database)

	// Note: We use a built-in driver named "postgres" registered via init()
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgresql connection: %w", err)
	}

	applyPoolSettings(db, config)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgresql database: %w", err)
	}

	return db, nil
}

func applyPoolSettings(db *sql.DB, config DatabaseConfig) {
	defaults := DefaultDatabaseConfig()
	db.SetMaxIdleConns(GetOrDefault(config.MaxIdleConns, defaults.MaxIdleConns))
	db.SetMaxOpenConns(GetOrDefault(config.MaxOpenConns, defaults.MaxOpenConns))
	db.SetConnMaxIdleTime(GetOrDefault(config.ConnIdleTime, defaults.ConnIdleTime))
	db.SetConnMaxLifetime(GetOrDefault(config.ConnMaxLifetime, defaults.ConnMaxLifetime))
}

// NewOracleFromConfig creates Oracle connection from application Config.
func NewOracleFromConfig(conf *Config) (*sql.DB, error) {
	oracle := conf.EnsureOracle()
	return NewOracleConnection(DatabaseConfig{
		Host:     oracle.Host,
		Port:     oracle.Port,
		Database: oracle.DatabaseName,
		Username: oracle.Username,
		Password: oracle.Password,
	})
}

// NewPostgreSQLFromConfig creates PostgreSQL connection from application Config.
func NewPostgreSQLFromConfig(conf *Config) (*sql.DB, error) {
	pg := conf.EnsurePostgreSQL()
	return NewPostgreSQLConnection(DatabaseConfig{
		Host:     pg.Host,
		Port:     pg.Port,
		Database: pg.DatabaseName,
		Username: pg.Username,
		Password: pg.Password,
	})
}

func parsePort(portStr string) (int, error) {
	if portStr == "" {
		return 0, nil
	}
	return strconv.Atoi(portStr)
}

// CloseDB safely closes a database connection.
func CloseDB(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// InterpolateQuery replaces named parameters in SQL query (FOR LOGGING ONLY).
func InterpolateQuery(query string, args ...any) string {
	if len(args) == 0 {
		return query
	}
	replacePairs := make([]string, 0, len(args)*2)
	for _, arg := range args {
		namedArg, ok := arg.(sql.NamedArg)
		if !ok {
			continue
		}
		replacePairs = append(replacePairs, ":"+namedArg.Name, formatValue(namedArg.Value))
	}
	if len(replacePairs) == 0 {
		return query
	}
	return strings.NewReplacer(replacePairs...).Replace(query)
}

func formatValue(v any) string {
	switch v := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

func LogInterpolatedQuery(query string, args ...any) {
	log.Printf("[SQL Query] %s", InterpolateQuery(query, args...))
}
