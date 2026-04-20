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

// LoggingData schedules a log message to be processed asynchronously.
// The output destination is determined by the LOG_OUTPUT_MODE environment variable.
func (le *LoggerEntry) LoggingData(level string, payload any, response ResponseLog) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response

	mode := os.Getenv("LOG_OUTPUT_MODE")
	if mode == "" {
		mode = "hybrid"
	}

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: le, payload: payload, mode: mode}:
		default:
			fmt.Printf("[WARNING] Log channel full, dropping log\n")
		}
	}
}

// SendToFile specifically queues the log to be written to a file.
func (le *LoggerEntry) SendToFile(level string, response ResponseLog) {
	le.Level = level
	le.Response = response
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: le, mode: "file"}:
		default:
		}
	}
}

// SendToElasticSearch specifically queues the log to be sent to Elasticsearch.
func (le *LoggerEntry) SendToElasticSearch(level string, payload any, response ResponseLog) {
	le.Level = level
	le.Response = response
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: le, payload: payload, mode: "elasticsearch"}:
		default:
		}
	}
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
