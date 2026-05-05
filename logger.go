// logger.go — Structured async logging for NanoPony.
//
// Architecture overview:
//
//	NewLoggerFromOptions()
//	   ↓
//	LoggerEntry.LoggingData(level, payload, response)
//	   ↓
//	logChan (buffered channel, cap=1000)
//	   ↓
//	background dispatcher goroutine
//	   ↓ dispatches to specialized sink workers (non-blocking):
//	   ├── consoleWorker       → stdout JSON
//	   ├── fileWorker          → rolling log file (lumberjack)
//	   └── esWorker            → ES index
//
// All logging is non-blocking. If a specific sink's channel is full, that 
// specific output is dropped to prevent blocking the entire logging pipeline.
package nanopony

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

const (
	// LogDir is the default directory for log files.
	LogDir = "./logs"
)

// LoggerEntry represents a structured log entry with rich metadata.
type LoggerEntry struct {
	mu              sync.RWMutex
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

// newLogger creates and returns a new structured logger entry (internal).
// It automatically handles working directory caching and internal worker initialization.
//
// Callers should use NewLoggerFromOptions instead.
func newLogger(
	serviceName, userLogin, referenceId, referenceNumber,
	systemName, processName, entity, additionals, nodeCode string,
) *LoggerEntry {
	onceCachedWd.Do(func() {
		cachedWd, _ = os.Getwd()
	})

	initLoggerFile()
	initLogWorker()

	return &LoggerEntry{
		StartTimestamp:  time.Now(),
		Service:         serviceName,
		UserLogin:       userLogin,
		ReferenceId:     referenceId,
		ReferenceNumber: referenceNumber,
		ProcessName:     processName,
		SystemName:      systemName,
		Entity:          entity,
		Additionals:     additionals,
		NodeCode:        nodeCode,
		Path:            cachedWd,
	}
}

// LoggerOptions provides named parameters for creating a LoggerEntry.
// This is the sole public constructor for LoggerEntry.
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
	NodeCode        string
}

// NewLoggerFromOptions creates a LoggerEntry from named options.
// This is the recommended and only public way to create loggers.
func NewLoggerFromOptions(opts LoggerOptions) *LoggerEntry {
	return newLogger(
		opts.ServiceName, opts.UserLogin, opts.ReferenceId, opts.ReferenceNumber,
		opts.SystemName, opts.ProcessName, opts.Entity, opts.Additionals, opts.NodeCode,
	)
}

// finalize sets the end timestamp, duration, level, and response on a log entry.
// Caller MUST hold le.mu.Lock() before calling.
func (le *LoggerEntry) finalize(level string, response ResponseLog) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response
}

// sendLog is the shared entry point for all log dispatching.
// It finalizes the entry, creates an immutable snapshot, and sends the snapshot
// to the async processing channel. This is safe for reusing the same LoggerEntry
// across multiple calls.
func (le *LoggerEntry) sendLog(level, mode string, payload any, response ResponseLog) {
	// Pre-process payload (Deep Copy) in the caller's goroutine to avoid data races
	// if the caller modifies the payload map/struct immediately after this call.
	processedPayload, _ := processPayload(payload)

	// Hold the lock while mutating AND cloning to prevent races when the same
	// LoggerEntry is reused across multiple LoggingData calls.
	le.mu.Lock()
	le.finalize(level, response)
	le.Request.Payload = processedPayload
	snapshot := le.cloneUnlocked()
	le.mu.Unlock()

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: snapshot, mode: mode}:
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
	mode := "hybrid"
	if conf := getAppConfig(); conf != nil && conf.App.LogOutputMode != "" {
		mode = conf.App.LogOutputMode
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

// LogFramework is a helper for internal framework logging.
func LogFramework(level, process, message string) {
	logger := newLogger("NanoPony", "system", "internal", "", "Framework", process, "System", "", "")
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
