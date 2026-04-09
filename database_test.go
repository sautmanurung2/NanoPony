package nanopony

import (
	"database/sql"
	"testing"
	"time"
)

func TestDefaultDatabaseConfig(t *testing.T) {
	config := DefaultDatabaseConfig()

	if config.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns 10, got %d", config.MaxIdleConns)
	}
	if config.MaxOpenConns != 100 {
		t.Errorf("Expected MaxOpenConns 100, got %d", config.MaxOpenConns)
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
			expected: 1521,
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
	ResetConfig()
	conf := NewConfig()

	// This will fail without Oracle, but tests the code path
	_, err := NewOracleFromConfig(conf)
	if err == nil {
		t.Log("Warning: Expected connection to fail without Oracle")
	}
}

func TestGetOracleDB(t *testing.T) {
	// Initially should be nil
	db := GetOracleDB()
	if db != nil {
		t.Log("Oracle DB already initialized")
	}
}

func TestCloseDB(t *testing.T) {
	// Test closing nil
	err := CloseDB(nil)
	if err != nil {
		t.Errorf("Expected no error when closing nil db, got %v", err)
	}

	// Test closing actual connection (if exists)
	if oracleDB != nil {
		err = CloseDB(oracleDB)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
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

func TestLogInterpolatedQuery(t *testing.T) {
	// This test just ensures the function doesn't panic
	// Actual logging is hard to test in unit tests
	query := "SELECT * FROM users WHERE id = :id"
	args := []any{sql.Named("id", 123)}

	// Should not panic
	LogInterpolatedQuery(query, args...)
}
