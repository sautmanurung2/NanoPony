// logger.go — Structured async logging for NanoPony.
//
// Architecture overview:
//
//	LogManager (Managed instance, no global state)
//	   ↓
//	NewLoggerFromOptions(manager)
//	   ↓
//	LoggerEntry.LoggingData(level, payload, response)
//	   ↓
//	logChan (buffered channel, managed by LogManager)
//	   ↓
//	background dispatcher goroutine
//	   ↓ dispatches to specialized sink workers (non-blocking):
//	   ├── consoleWorker       → stdout JSON
//	   ├── fileWorker          → rolling log file (lumberjack)
//	   └── esWorker            → ES index
//
// Beginner Note: We use a "Manager" pattern here to avoid global variables.
// Global variables (global state) make code harder to test and can cause
// unexpected behavior when multiple parts of a program try to use the same logger.
package nanopony

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/natefinch/lumberjack"
)

const (
	// LogDir is the default directory for log files.
	LogDir = "./logs"
)

// LogManager orchestrates background logging workers and resources.
// It owns the channels and lifecycle of the logging system.
type LogManager struct {
	config        *Config
	logChan       chan logRequest
	consoleChan   chan logRequest
	fileChan      chan logRequest
	esChan        chan logRequest
	esClient      *elasticsearch.Client
	logFileWriter *lumberjack.Logger
	wg            sync.WaitGroup
	stopOnce      sync.Once
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewLogManager creates a new instance of the logging system.
func NewLogManager(conf *Config) *LogManager {
	ctx, cancel := context.WithCancel(context.Background())
	lm := &LogManager{
		config:      conf,
		logChan:     make(chan logRequest, 1000),
		consoleChan: make(chan logRequest, 1000),
		fileChan:    make(chan logRequest, 1000),
		esChan:      make(chan logRequest, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}

	lm.initWorkers()
	return lm
}

// Shutdown gracefully stops all logging workers and flushes remaining logs.
func (lm *LogManager) Shutdown() {
	lm.stopOnce.Do(func() {
		lm.cancel()
		close(lm.logChan)
		lm.wg.Wait()
		
		if lm.logFileWriter != nil {
			_ = lm.logFileWriter.Close()
		}
	})
}

// LoggerEntry represents a structured log entry with rich metadata.
type LoggerEntry struct {
	manager         *LogManager
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

// NewLoggerFromOptions creates a LoggerEntry from named options.
// Beginner Note: The manager parameter is an example of "Dependency Injection".
// We pass the manager to the logger so it knows where to send its data.
func NewLoggerFromOptions(manager *LogManager, opts LoggerOptions) *LoggerEntry {
	wd, _ := os.Getwd()
	return &LoggerEntry{
		manager:         manager,
		StartTimestamp:  time.Now(),
		Service:         opts.ServiceName,
		UserLogin:       opts.UserLogin,
		ReferenceId:     opts.ReferenceId,
		ReferenceNumber: opts.ReferenceNumber,
		ProcessName:     opts.ProcessName,
		SystemName:      opts.SystemName,
		Entity:          opts.Entity,
		Additionals:     opts.Additionals,
		NodeCode:        opts.NodeCode,
		Path:            wd,
	}
}

// LoggerOptions provides named parameters for creating a LoggerEntry.
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

// finalize sets the end timestamp, duration, level, and response on a log entry.
func (le *LoggerEntry) finalize(level string, response ResponseLog) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response
}

// sendLog is the shared entry point for all log dispatching.
func (le *LoggerEntry) sendLog(level, mode string, payload any, response ResponseLog) {
	processedPayload := processPayload(payload)

	le.mu.Lock()
	le.finalize(level, response)
	le.Request.Payload = processedPayload
	snapshot := le.cloneUnlocked()
	le.mu.Unlock()

	if le.manager != nil && le.manager.logChan != nil {
		select {
		case le.manager.logChan <- logRequest{entry: snapshot, mode: mode}:
		default:
			// Non-blocking drop if channel is full
		}
	}
}

// LoggingData schedules a log message to be processed asynchronously.
func (le *LoggerEntry) LoggingData(level string, payload any, response ResponseLog) {
	mode := "hybrid"
	if le.manager != nil && le.manager.config != nil && le.manager.config.App.LogOutputMode != "" {
		mode = le.manager.config.App.LogOutputMode
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
func (lm *LogManager) LogFramework(level, process, message string) {
	opts := LoggerOptions{
		ServiceName: "NanoPony",
		UserLogin:   "system",
		ReferenceId: "internal",
		SystemName:  "Framework",
		ProcessName: process,
		Entity:      "System",
	}
	logger := NewLoggerFromOptions(lm, opts)
	logger.LoggingData(level, nil, ResponseLog{Status: level, Message: message})
}
