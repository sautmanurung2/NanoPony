package nanopony

import (
	"encoding/json"
	"testing"
)

func TestNewLogManager(t *testing.T) {
	ResetConfig()
	config := NewConfig()
	lm := NewLogManager(config)
	if lm == nil {
		t.Fatal("Expected LogManager to be created")
	}
	defer lm.Shutdown()
}

func TestNewLogger(t *testing.T) {
	ResetConfig()
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()

	logger := NewLoggerFromOptions(lm, LoggerOptions{
		ServiceName:     "TestService",
		UserLogin:       "testuser",
		ReferenceId:     "ref-001",
		ReferenceNumber: "ref-num-001",
		SystemName:      "TestSystem",
		ProcessName:     "TestProcess",
		Entity:          "TestEntity",
		Additionals:     "additional info",
	})

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
	if logger.Service != "TestService" {
		t.Errorf("Expected service 'TestService', got '%s'", logger.Service)
	}
}

func TestLoggerEntry(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()

	logger := NewLoggerFromOptions(lm, LoggerOptions{ServiceName: "TestService"})

	response := ResponseLog{
		Status:  "success",
		Message: "Test message",
	}

	// Should not panic
	logger.SendToFile("INFO", response)
}

func TestProcessPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{"map", map[string]any{"key": "value"}},
		{"string", `{"key": "value"}`},
		{"struct", struct{ Name string }{Name: "test"}},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processPayload(tt.payload)
			if tt.payload != nil && result == nil {
				t.Error("Expected result map, got nil")
			}
		})
	}
}

func TestLogManagerFileName(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()

	fileName := lm.generateLogFileName()
	if fileName == "" {
		t.Error("Expected non-empty filename")
	}
}

func TestLoggerEntryWriteToConsole(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()
	logger := NewLoggerFromOptions(lm, LoggerOptions{})

	// Should not panic
	logger.writeToConsole()
}

func TestLoggerEntryLoggingData(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()
	logger := NewLoggerFromOptions(lm, LoggerOptions{})

	payload := map[string]any{"key": "value"}
	response := ResponseLog{Status: "success", Message: "Test"}

	logger.LoggingData("INFO", payload, response)

	if logger.Level != "INFO" {
		t.Errorf("Expected level 'INFO', got '%s'", logger.Level)
	}
}

func TestLoggerEntryJSONMarshaling(t *testing.T) {
	logger := &LoggerEntry{
		ReferenceId: "ref-001",
		Level:       "INFO",
	}

	data, err := json.Marshal(logger)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled LoggerEntry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ReferenceId != logger.ReferenceId {
		t.Errorf("Expected '%s', got '%s'", logger.ReferenceId, unmarshaled.ReferenceId)
	}
}

func TestLoggerInternalClones(t *testing.T) {
	le := &LoggerEntry{ReferenceId: "test"}
	c1 := le.clone()
	if c1.ReferenceId != "test" {
		t.Error("clone failed")
	}
}

func TestDeepCopyMapEdgeCases(t *testing.T) {
	m1 := map[string]any{"a": map[string]any{"b": 1}}
	m2 := deepCopyMap(m1, 0)
	if m2["a"].(map[string]any)["b"] != 1 {
		t.Error("deepCopyMap failed")
	}
}

func TestLogFrameworkMethod(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()
	lm.LogFramework("INFO", "Test", "Message")
}

func TestWriteToElasticsearchNoConfig(t *testing.T) {
	lm := NewLogManager(NewConfig())
	defer lm.Shutdown()
	logger := NewLoggerFromOptions(lm, LoggerOptions{})
	
	// Should not panic even if ES not configured
	logger.writeToElasticsearch(lm)
}
