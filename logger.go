// logger.go — Structured async logging for NanoPony.
//
// Architecture overview:
//
//	NewLogger()
//	   ↓
//	LoggerEntry.LoggingData(level, payload, response)
//	   ↓
//	logChan (buffered channel, cap=1000)
//	   ↓
//	background worker goroutine
//	   ↓ dispatches by mode:
//	   ├── "console"       → stdout JSON
//	   ├── "file"          → rolling log file (lumberjack)
//	   ├── "elasticsearch" → ES index
//	   └── "hybrid" (default) → console + file + ES
//
// All logging is non-blocking. If the channel is full, the log is dropped
// with a warning printed to stdout.
package nanopony

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

const (
	// LogDir is the default directory for log files.
	LogDir = "./logs"
)

// LoggerEntry represents a structured log entry with rich metadata.
type LoggerEntry struct {
	StartTimestamp  time.Time   `json:"start_timestamp"`
	EndTimestamp    time.Time   `json:"end_timestamp"`
	ReferenceId     string      `json:"reference_id"`
	ReferenceNumber string      `json:"reference_number"`
	ProcessName     string      `json:"process_name"`
	SystemName      string      `json:"system_name"`
	Entity          string      `json:"entity"`
	Additionals     string      `json:"additionals"`
	Duration        int64       `json:"duration"`
	Service         string      `json:"service"`
	Path            string      `json:"path"`
	Level           string      `json:"level"`
	UserLogin       string      `json:"user_login"`
	NodeCode        string      `json:"node_code"`
	Request         RequestLog  `json:"request"`
	Response        ResponseLog `json:"response"`
}

// RequestLog represents the payload portion of a log entry.
type RequestLog struct {
	Payload map[string]any `json:"payload"`
}

// ResponseLog represents the outcome portion of a log entry.
type ResponseLog struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewLogger creates and returns a new structured logger entry.
// It automatically handles working directory caching and internal worker initialization.
//
// For a more readable alternative with named parameters, see NewLoggerFromOptions.
func NewLogger(
	serviceName, userLogin, referenceId, referenceNumber,
	systemName, processName, entity, additionals string,
) *LoggerEntry {
	onceCachedWd.Do(func() {
		cachedWd, _ = os.Getwd()
	})

	initLoggerFile()
	initLogWorker()

	return &LoggerEntry{
		StartTimestamp:  time.Now(),
		Service:         "GO_" + serviceName,
		UserLogin:       userLogin,
		ReferenceId:     referenceId,
		ReferenceNumber: referenceNumber,
		ProcessName:     processName,
		SystemName:      "GO-Producer-" + systemName,
		Entity:          entity,
		Additionals:     additionals,
		Path:            cachedWd,
	}
}

// LoggerOptions provides named parameters for creating a LoggerEntry.
// This is easier to read than NewLogger's 8 positional string parameters.
//
// Example:
//
//	logger := nanopony.NewLoggerFromOptions(nanopony.LoggerOptions{
//	    ServiceName: "OrderService",
//	    UserLogin:   "admin",
//	    SystemName:  "Core",
//	    ProcessName: "ProcessOrder",
//	    Entity:      "Order",
//	})
type LoggerOptions struct {
	ServiceName     string
	UserLogin       string
	ReferenceId     string
	ReferenceNumber string
	SystemName      string
	ProcessName     string
	Entity          string
	Additionals     string
}

// NewLoggerFromOptions creates a LoggerEntry from named options.
// This is the recommended way to create loggers in new code.
func NewLoggerFromOptions(opts LoggerOptions) *LoggerEntry {
	return NewLogger(
		opts.ServiceName, opts.UserLogin, opts.ReferenceId, opts.ReferenceNumber,
		opts.SystemName, opts.ProcessName, opts.Entity, opts.Additionals,
	)
}

// finalize sets the end timestamp, duration, level, and response on a log entry.
func (le *LoggerEntry) finalize(level string, response ResponseLog) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response
}

// sendLog is the shared entry point for all log dispatching.
// It finalizes the entry and sends it to the async processing channel.
func (le *LoggerEntry) sendLog(level, mode string, payload any, response ResponseLog) {
	le.finalize(level, response)

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: le, payload: payload, mode: mode}:
		default:
			fmt.Printf("[WARNING] Log channel full, dropping log\n")
		}
	}
}

// LoggingData schedules a log message to be processed asynchronously.
// The output destination is determined by the LOG_OUTPUT_MODE environment variable:
//
//	"console"       — print JSON to stdout
//	"file"          — write to rolling log file
//	"elasticsearch" — send to Elasticsearch index
//	"hybrid"        — all three (default if LOG_OUTPUT_MODE is not set)
func (le *LoggerEntry) LoggingData(level string, payload any, response ResponseLog) {
	mode := os.Getenv("LOG_OUTPUT_MODE")
	if mode == "" {
		mode = "hybrid"
	}
	le.sendLog(level, mode, payload, response)
}

// SendToFile specifically queues the log to be written to a file.
func (le *LoggerEntry) SendToFile(level string, response ResponseLog) {
	le.sendLog(level, "file", nil, response)
}

// SendToElasticSearch specifically queues the log to be sent to Elasticsearch.
func (le *LoggerEntry) SendToElasticSearch(level string, payload any, response ResponseLog) {
	le.sendLog(level, "elasticsearch", payload, response)
}

// LogInternal is a helper for logging internal messages using an existing LoggerEntry.
func (le *LoggerEntry) LogInternal(level, process, message string) {
	le.ProcessName = process
	le.LoggingData(level, nil, ResponseLog{
		Status:  level,
		Message: message,
	})
}

// LogFramework is a helper for internal framework logging.
func LogFramework(level, process, message string) {
	logger := NewLogger("NanoPony", "system", "internal", "", "Framework", process, "System", "")
	logger.LoggingData(level, nil, ResponseLog{Status: level, Message: message})
}

// InitElasticsearch initializes and tests the Elasticsearch client connection.
func InitElasticsearch() (*elasticsearch.Client, error) {
	conf := getAppConfig()
	if conf == nil {
		return nil, fmt.Errorf("application config not initialized")
	}

	es := conf.EnsureElasticSearch()
	cfg := elasticsearch.Config{
		Addresses: []string{es.ElasticHost},
		Username:  es.ElasticUsername,
		Password:  es.ElasticPassword,
		APIKey:    es.ElasticApiKey,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if _, err := client.Info(); err != nil {
		return nil, fmt.Errorf("ES ping failed: %w", err)
	}

	return client, nil
}
