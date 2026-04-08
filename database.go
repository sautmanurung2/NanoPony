package nanopony

import (
	"database/sql"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

var oracleDB *sql.DB

// DatabaseConfig holds Oracle database connection configuration
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

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnIdleTime:    5 * time.Minute,
		ConnMaxLifetime: 60 * time.Minute,
	}
}

// NewOracleConnection creates a new Oracle database connection
func NewOracleConnection(config DatabaseConfig) (*sql.DB, error) {
	port := 1521
	if config.Port != "" {
		if _, err := fmt.Sscanf(config.Port, "%d", &port); err != nil {
			port = 1521
		}
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

	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetConnMaxIdleTime(config.ConnIdleTime)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping oracle database: %w", err)
	}

	oracleDB = db

	return db, nil
}

// NewOracleFromConfig creates Oracle connection from Config
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

func GetOracleDB() *sql.DB {
	return oracleDB
}

// CloseDB safely closes a database connection
func CloseDB(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}
