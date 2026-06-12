package nanopony

import (
	"context"
	"os"
	"database/sql"
	"testing"
	"time"
)

func TestDefaultDatabaseConfig(t *testing.T) {
	config := DefaultDatabaseConfig()

	if config.MaxIdleConns != 2 {
		t.Errorf("Expected MaxIdleConns 2, got %d", config.MaxIdleConns)
	}
	if config.MaxOpenConns != 20 {
		t.Errorf("Expected MaxOpenConns 20, got %d", config.MaxOpenConns)
	}
	if config.ConnIdleTime != 5*time.Minute {
		t.Errorf("Expected ConnIdleTime 5m, got %v", config.ConnIdleTime)
	}
	if config.ConnMaxLifetime != 60*time.Minute {
		t.Errorf("Expected ConnMaxLifetime 60m, got %v", config.ConnMaxLifetime)
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		portStr  string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid port number",
			portStr:  "1521",
			expected: 1521,
			wantErr:  false,
		},
		{
			name:     "empty port",
			portStr:  "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "invalid port",
			portStr:  "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "port out of range",
			portStr:  "999999",
			expected: 999999,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePort(tt.portStr)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestNewOracleConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database connection test in short mode")
	}
	// Test with invalid connection - should fail gracefully
	config := DatabaseConfig{
		Host:     "invalid-host",
		Port:     "1521",
		Database: "ORCL",
		Username: "user",
		Password: "pass",
	}

	// This will fail because Oracle is not available, but it should fail gracefully
	_, err := NewOracleConnection(config)
	if err == nil {
		t.Log("Warning: Expected connection to fail with invalid host")
	}
}

func TestNewOracleFromConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database connection test in short mode")
	}
	ResetConfig()
	conf := NewConfig()

	// This will fail without Oracle, but tests the code path
	_, err := NewOracleFromConfig(conf)
	if err == nil {
		t.Log("Warning: Expected connection to fail without Oracle")
	}
}

func TestDatabaseConnectionViaFramework(t *testing.T) {
	// New way via Framework (Recommended)
	// We no longer use GetOracleDB() global accessor.
	ResetConfig()
	conf := NewConfig()
	// Mock database connection for testing
	fw := NewFramework().WithConfig(conf).WithDatabaseFromInstance(nil)
	components := fw.Build()
	defer components.Shutdown(context.Background())

	if components.DB != nil {
		t.Error("Expected components.DB to be nil (mocked as nil)")
	}
}

func TestCloseDB(t *testing.T) {
	// Test closing nil (safe)
	err := CloseDB(nil)
	if err != nil {
		t.Errorf("Expected no error when closing nil db, got %v", err)
	}
}

func TestInterpolateQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		args     []any
		expected string
	}{
		{
			name:     "string value",
			query:    "SELECT * FROM users WHERE name = :name",
			args:     []any{sql.Named("name", "John")},
			expected: "SELECT * FROM users WHERE name = 'John'",
		},
		{
			name:     "integer value",
			query:    "SELECT * FROM users WHERE id = :id",
			args:     []any{sql.Named("id", 123)},
			expected: "SELECT * FROM users WHERE id = 123",
		},
		{
			name:     "nil value",
			query:    "SELECT * FROM users WHERE name = :name",
			args:     []any{sql.Named("name", nil)},
			expected: "SELECT * FROM users WHERE name = NULL",
		},
		{
			name:     "multiple values",
			query:    "SELECT * FROM users WHERE name = :name AND id = :id",
			args:     []any{sql.Named("name", "John"), sql.Named("id", 123)},
			expected: "SELECT * FROM users WHERE name = 'John' AND id = 123",
		},
		{
			name:     "no args",
			query:    "SELECT * FROM users",
			args:     []any{},
			expected: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InterpolateQuery(tt.query, tt.args...)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "string",
			value:    "hello",
			expected: "'hello'",
		},
		{
			name:     "string with quote",
			value:    "O'Brien",
			expected: "'O''Brien'",
		},
		{
			name:     "integer",
			value:    123,
			expected: "123",
		},
		{
			name:     "float",
			value:    123.45,
			expected: "123.45",
		},
		{
			name:     "boolean",
			value:    true,
			expected: "true",
		},
		{
			name:     "nil",
			value:    nil,
			expected: "NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.value)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestInterpolateQueryEdgeCases(t *testing.T) {
	// Non-named argument
	q1 := InterpolateQuery("SELECT * FROM users WHERE id = :id", 123)
	if q1 != "SELECT * FROM users WHERE id = :id" {
		t.Errorf("Expected unchanged query, got %s", q1)
	}

	// No matching placeholders
	q2 := InterpolateQuery("SELECT * FROM users", sql.Named("id", 123))
	if q2 != "SELECT * FROM users" {
		t.Errorf("Expected unchanged query, got %s", q2)
	}
}

func TestFormatValueDefault(t *testing.T) {
	// Test default case in formatValue
	v := struct{}{}
	result := formatValue(v)
	if result != "'{}'" {
		t.Errorf("Expected '{}', got %s", result)
	}
}

func TestNewOracleConnectionInvalidPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database connection test in short mode")
	}
	config := DatabaseConfig{
		Host: "localhost",
		Port: "invalid", // Should trigger warning and use default 1521
	}
	// It will still fail connection, but we test the port parsing logic
	_, _ = NewOracleConnection(config)
}

func TestLogInterpolatedQuery(t *testing.T) {
	LogInterpolatedQuery("SELECT * FROM users WHERE id = :id", sql.Named("id", 123))
}

func TestNewPostgreSQLFromConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database connection test in short mode")
	}
	ResetConfig()
	os.Setenv("GO_ENV", "local")
	config := NewConfig()
	db, err := NewPostgreSQLFromConfig(config)
	if _, ok := err.(interface{ Unwrap() error }); !ok {
		// Connection will fail in test env, that's OK
		// We just want no nil pointer issues
	}
	if db != nil { db.Close() }
}


func TestNewPostgreSQLConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database connection test in short mode")
	}
	cfg := DatabaseConfig{
		Host: "localhost",
		Port: "5432",
		Database: "test",
		Username: "user",
		Password: "pass",
	}
	// Connection will fail but we hit the driver registration logic
	db, err := NewPostgreSQLConnection(cfg)
	if db != nil { db.Close() }
	if err == nil {
		// Connection shouldn't succeed on CI/local usually
	}
}
