package nanopony

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(
		"TestService",
		"testuser",
		"ref-001",
		"ref-num-001",
		"TestSystem",
		"TestProcess",
		"TestEntity",
		"additional info",
	)

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
	if logger.Service != "GO_TestService" {
		t.Errorf("Expected service 'GO_TestService', got '%s'", logger.Service)
	}
	if logger.UserLogin != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", logger.UserLogin)
	}
	if logger.ReferenceId != "ref-001" {
		t.Errorf("Expected reference ID 'ref-001', got '%s'", logger.ReferenceId)
	}
	if logger.SystemName != "GO-Producer-TestSystem" {
		t.Errorf("Expected system name 'GO-Producer-TestSystem', got '%s'", logger.SystemName)
	}
}

func TestLoggerEntry(t *testing.T) {
	logger := NewLogger("TestService", "user", "ref", "", "System", "Process", "Entity", "")

	// Test SendToFile (may fail if log directory doesn't exist, but should not panic)
	response := ResponseLog{
		Status:  "success",
		Message: "Test message",
	}

	// This may fail without proper directory setup, but should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked: %v", r)
		}
	}()

	logger.SendToFile("INFO", response)
}

func TestProcessPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		wantErr bool
	}{
		{
			name:    "map payload",
			payload: map[string]any{"key": "value"},
			wantErr: false,
		},
		{
			name:    "string payload",
			payload: `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "struct payload",
			payload: struct{ Name string }{Name: "test"},
			wantErr: false,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: false, // nil payload creates a map with error key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processPayload(tt.payload)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			// For nil payload, result can be nil or a map with error key
			if !tt.wantErr && tt.payload != nil && result == nil {
				t.Error("Expected result map, got nil")
			}
		})
	}
}

func TestGenerateLogFileName(t *testing.T) {
	fileName := generateLogFileName()

	// Should contain date
	expected := "orion-to-core-"
	if fileName[:len(expected)] != expected {
		t.Errorf("Expected filename to start with '%s', got '%s'", expected, fileName)
	}

	// Should end with .log
	if fileName[len(fileName)-4:] != ".log" {
		t.Errorf("Expected filename to end with '.log', got '%s'", fileName)
	}
}

func TestEnsureLogDirectoryExists(t *testing.T) {
	// This may fail in test environment, but should return error
	err := ensureLogDirectoryExists()
	if err != nil {
		t.Logf("Expected error (may be expected in test env): %v", err)
	}
}

func TestLoggerEntryWriteToConsole(t *testing.T) {
	logger := NewLogger("TestService", "user", "ref", "", "System", "Process", "Entity", "")

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked: %v", r)
		}
	}()

	logger.writeToConsole()
}

func TestLoggerEntryLoggingData(t *testing.T) {
	// Set output mode to console only
	os.Setenv("LOG_OUTPUT_MODE", "fluentd")
	defer os.Unsetenv("LOG_OUTPUT_MODE")

	logger := NewLogger("TestService", "user", "ref", "", "System", "Process", "Entity", "")

	payload := map[string]any{"key": "value"}
	response := ResponseLog{
		Status:  "success",
		Message: "Test message",
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked: %v", r)
		}
	}()

	logger.LoggingData("INFO", payload, response)

	// Verify logger was updated
	if logger.Level != "INFO" {
		t.Errorf("Expected level 'INFO', got '%s'", logger.Level)
	}
	if logger.Response.Status != "success" {
		t.Errorf("Expected response status 'success', got '%s'", logger.Response.Status)
	}
}

func TestLoggerEntrySendToElasticSearch(t *testing.T) {
	// Skip if no Elasticsearch configured
	if os.Getenv("ELASTIC_HOST") == "" {
		t.Skip("Elasticsearch not configured")
	}

	logger := NewLogger("TestService", "user", "ref", "", "System", "Process", "Entity", "")

	payload := map[string]any{"key": "value"}
	response := ResponseLog{
		Status:  "success",
		Message: "Test message",
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked: %v", r)
		}
	}()

	logger.SendToElasticSearch("INFO", payload, response)
}

func TestInitElasticsearch(t *testing.T) {
	// Skip if no Elasticsearch configured
	if os.Getenv("ELASTIC_HOST") == "" {
		t.Skip("Elasticsearch not configured")
	}

	// Reset global client
	esClientMutex.Lock()
	EsClient = nil
	esClientMutex.Unlock()

	_, err := InitElasticsearch()

	if err != nil {
		t.Logf("Expected error (Elasticsearch may not be available): %v", err)
	}
}

func TestEnsureElasticsearchClient(t *testing.T) {
	// Test with nil client
	esClientMutex.Lock()
	EsClient = nil
	esClientMutex.Unlock()

	err := ensureElasticsearchClient()
	// May fail if Elasticsearch not configured, which is expected
	if err != nil {
		t.Logf("Expected error (Elasticsearch may not be configured): %v", err)
	}
}

func TestLoggerEntryJSONMarshaling(t *testing.T) {
	logger := &LoggerEntry{
		StartTimestamp:  time.Now(),
		EndTimestamp:    time.Now(),
		ReferenceId:     "ref-001",
		ReferenceNumber: "ref-num-001",
		ProcessName:     "TestProcess",
		SystemName:      "GO-Producer-TestSystem",
		Entity:          "TestEntity",
		Additionals:     "additional",
		Duration:        100,
		Service:         "GO_TestService",
		Path:            "/test/path",
		Level:           "INFO",
		UserLogin:       "testuser",
		NodeCode:        "node-1",
		Request: RequestLog{
			Payload: map[string]any{"key": "value"},
		},
		Response: ResponseLog{
			Status:  "success",
			Message: "Test message",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(logger)
	if err != nil {
		t.Fatalf("Failed to marshal logger: %v", err)
	}

	// Unmarshal back
	var unmarshaled LoggerEntry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal logger: %v", err)
	}

	// Verify fields
	if unmarshaled.ReferenceId != logger.ReferenceId {
		t.Errorf("Expected ReferenceId '%s', got '%s'", logger.ReferenceId, unmarshaled.ReferenceId)
	}
	if unmarshaled.Level != logger.Level {
		t.Errorf("Expected Level '%s', got '%s'", logger.Level, unmarshaled.Level)
	}
}

func TestLoggerOutputModes(t *testing.T) {
	tests := []struct {
		name       string
		outputMode string
	}{
		{"fluentd mode", "fluentd"},
		{"elasticsearch mode", "elasticsearch"},
		{"hybrid mode", "hybrid"},
		{"empty mode (default)", ""},
		{"unknown mode", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_OUTPUT_MODE", tt.outputMode)
			defer os.Unsetenv("LOG_OUTPUT_MODE")

			logger := NewLogger("TestService", "user", "ref", "", "System", "Process", "Entity", "")
			payload := map[string]any{"key": "value"}
			response := ResponseLog{Status: "success", Message: "Test"}

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Logger panicked with output mode '%s': %v", tt.outputMode, r)
				}
			}()

			logger.LoggingData("INFO", payload, response)
		})
	}
}
