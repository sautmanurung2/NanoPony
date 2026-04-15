package nanopony

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
	"github.com/natefinch/lumberjack"
)

const (
	// LogDir is the default directory for log files.
	LogDir = "./logs"
)

var (
	// EsClient is the global Elasticsearch client instance.
	// This is set automatically when logging to Elasticsearch.
	// Protected by esClientMutex for thread safety.
	EsClient      *elasticsearch.Client
	esClientMutex sync.RWMutex
	_             = godotenv.Load()

	logFileWriter *lumberjack.Logger
	onceLogger    sync.Once

	// logChan is the buffered channel for asynchronous logging.
	logChan chan logRequest
	// onceWorker ensures the log worker is started only once.
	onceWorker sync.Once
)

// logRequest represents a request to log data asynchronously.
type logRequest struct {
	entry    *LoggerEntry
	payload  any
	response ResponseLog
	mode     string // "file", "console", "elasticsearch", "hybrid"
}

// initLogWorker starts the background goroutine that processes log requests.
func initLogWorker() {
	onceWorker.Do(func() {
		// Buffer size of 1000 to handle bursts of logs
		logChan = make(chan logRequest, 1000)
		go func() {
			for req := range logChan {
				processLogRequest(req)
			}
		}()
	})
}

// processLogRequest performs the actual I/O for a log request.
func processLogRequest(req logRequest) {
	switch req.mode {
	case "console":
		req.entry.printToConsole()
	case "file":
		req.entry.sendToFileNow(req.response)
	case "elasticsearch":
		req.entry.sendToElasticSearchNow(req.entry.Level, req.payload, req.response)
	case "hybrid":
		req.entry.printToConsole()
		req.entry.sendToFileNow(req.response)
		req.entry.sendToElasticSearchNow(req.entry.Level, req.payload, req.response)
	}
}

// initLoggerFile initializes the log file writer (singleton pattern).
// It creates a lumberjack logger with rotation settings.
func initLoggerFile() {
	onceLogger.Do(func() {
		if err := ensureLogDirectoryExists(); err != nil {
			fmt.Printf("Failed to create log directory: %s\n", err)
			return
		}

		logFileName := generateLogFileName()
		logFilePath := path.Join(LogDir, logFileName)

		logFileWriter = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100, // megabytes
			MaxBackups: 3,   // number of old files to keep
			MaxAge:     28,  // days
			Compress:   false,
		}
	})
}

// LoggerEntry represents a structured log entry with rich metadata.
// It supports multiple output modes: console, file, Elasticsearch, or hybrid.
//
// Example:
//
//	logger := NewLogger("MyService", "user123", "ref-001", "", "System", "ProcessData", "User", "")
//	logger.LoggingData("INFO", requestData, ResponseLog{Status: "success", Message: "Processed"})
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

// RequestLog represents the request portion of a log entry
type RequestLog struct {
	Payload map[string]any `json:"payload"`
}

// ResponseLog represents the response portion of a log entry
type ResponseLog struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewLogger creates a new structured logger entry.
//
// Parameters:
//   - serviceName: Name of the service (will be prefixed with "GO_")
//   - userLogin: User who triggered the operation
//   - referenceId: Unique reference ID for tracking
//   - referenceNumber: Additional reference number
//   - systemName: System name (will be prefixed with "GO-Producer-")
//   - processName: Name of the process
//   - entity: Entity being processed
//   - additionals: Additional information
//
// Example:
//
//	logger := NewLogger(
//	    "UserService",
//	    "admin",
//	    "req-123",
//	    "",
//	    "CoreSystem",
//	    "ProcessUser",
//	    "User",
//	    "",
//	)
func NewLogger(
	serviceName string,
	userLogin string,
	referenceId string,
	referenceNumber string,
	systemName string,
	processName string,
	entity string,
	additionals string,
) *LoggerEntry {
	dir, _ := os.Getwd()
	serviceName = "GO_" + serviceName
	systemNames := "GO-Producer-" + systemName

	initLoggerFile()
	initLogWorker() // Start async log worker

	loggerEntry := &LoggerEntry{
		StartTimestamp:  time.Now(),
		Service:         serviceName,
		UserLogin:       userLogin,
		ReferenceId:     referenceId,
		ReferenceNumber: referenceNumber,
		ProcessName:     processName,
		SystemName:      systemNames,
		Entity:          entity,
		Additionals:     additionals,
		Path:            dir, // Working directory removed - use LOG_NODE_CODE instead
		NodeCode:        "",
	}

	return loggerEntry
}

// SendToFile queues the log entry to be written to a rotating log file asynchronously.
func (le *LoggerEntry) SendToFile(level string, response ResponseLog) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response

	if logChan != nil {
		select {
		case logChan <- logRequest{entry: le, response: response, mode: "file"}:
		default:
			fmt.Fprintf(os.Stderr, "[WARNING] Log channel full, dropping log to file\n")
		}
	}
}

// sendToFileNow writes the log entry to a rotating log file immediately.
func (le *LoggerEntry) sendToFileNow(response ResponseLog) {
	logMessage, _ := json.Marshal(le)

	if logFileWriter == nil {
		fmt.Fprintf(os.Stderr, "Log file writer not initialized\n")
		return
	}

	_, err := fmt.Fprintf(logFileWriter, "%s\n", string(logMessage))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
	}
}

// LoggingData queues log data to be processed asynchronously based on LOG_OUTPUT_MODE.
func (le *LoggerEntry) LoggingData(
	level string,
	payload any,
	response ResponseLog,
) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response

	outputMode := os.Getenv("LOG_OUTPUT_MODE")
	if outputMode == "" {
		outputMode = "hybrid"
	}

	if logChan != nil {
		select {
		case logChan <- logRequest{
			entry:    le,
			payload:  payload,
			response: response,
			mode:     outputMode,
		}:
		default:
			fmt.Printf("[WARNING] Log channel full, dropping log (mode: %s)\n", outputMode)
		}
	} else {
		// Fallback if worker not started
		processLogRequest(logRequest{
			entry:    le,
			payload:  payload,
			response: response,
			mode:     outputMode,
		})
	}
}

// processPayload converts various payload types into a map for JSON logging
func processPayload(payload any) (map[string]any, error) {
	var payloadMap map[string]any

	switch v := payload.(type) {
	case map[string]any:
		payloadMap = v

	case string, []byte:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", v)), &parsed); err == nil {
			payloadMap = parsed
		} else {
			payloadMap = map[string]any{
				"raw": fmt.Sprintf("%v", v),
			}
		}

	default:
		dataBytes, err := json.Marshal(v)
		if err != nil {
			payloadMap = map[string]any{
				"error": "failed_to_marshal",
				"raw":   fmt.Sprintf("%v", v),
			}
		} else {
			if err := json.Unmarshal(dataBytes, &payloadMap); err != nil {
				payloadMap = map[string]any{
					"error": "failed_to_unmarshal_after_marshal",
					"raw":   string(dataBytes),
				}
			}
		}
	}

	return payloadMap, nil
}

// generateLogFileName creates a log filename with current date
func generateLogFileName() string {
	currentTime := time.Now()
	year := currentTime.Year()
	month := int(currentTime.Month())
	day := currentTime.Day()

	return fmt.Sprintf("orion-to-core-%d-%02d-%02d.log", year, month, day)
}

// ensureLogDirectoryExists creates the log directory if it doesn't exist.
// Returns an error if directory creation fails.
func ensureLogDirectoryExists() error {
	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		if err := os.Mkdir(LogDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	}
	return nil
}

// printToConsole prints the log entry to console as JSON
func (le *LoggerEntry) printToConsole() {
	data, err := json.Marshal(le)
	if err != nil {
		fmt.Printf("Error marshaling for console: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

// SendToElasticSearch queues the log entry to be sent to Elasticsearch asynchronously.
func (le *LoggerEntry) SendToElasticSearch(
	level string,
	payload any,
	response ResponseLog,
) {
	if logChan != nil {
		select {
		case logChan <- logRequest{
			entry:    le,
			payload:  payload,
			response: response,
			mode:     "elasticsearch",
		}:
		default:
			fmt.Printf("[WARNING] Log channel full, dropping log to Elasticsearch\n")
		}
	}
}

// sendToElasticSearchNow sends the log entry to Elasticsearch immediately.
func (le *LoggerEntry) sendToElasticSearchNow(
	level string,
	payload any,
	response ResponseLog,
) {
	// Initialize Elasticsearch client if needed
	if err := ensureElasticsearchClient(); err != nil {
		fmt.Printf("Failed to initialize Elasticsearch client: %v\n", err)
		return
	}

	esClient := EsClient
	payloadMap, err := processPayload(payload)
	if err != nil {
		fmt.Printf("Error processing payload: %v\n", err)
		return
	}

	le.Request.Payload = payloadMap
	data, err := json.Marshal(le)
	if err != nil {
		fmt.Printf("Error marshaling message to JSON: %v\n", err)
		return
	}

	esIndexWithDate := appConfig.ElasticSearch.ElasticPrefixIndex + time.Now().Format("20060102")
	esResult, err := esClient.Index(esIndexWithDate, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Error indexing message to Elasticsearch: %v\n", err)
		return
	}
	defer esResult.Body.Close()
}

// LogFramework is a global helper for internal framework logging.
// It creates a generic logger entry for system-level messages.
func LogFramework(level, process, message string) {
	logger := NewLogger(
		"NanoPony",
		"system",
		"internal",
		"",
		"Framework",
		process,
		"System",
		"",
	)
	logger.LoggingData(level, nil, ResponseLog{
		Status:  level,
		Message: message,
	})
}

// LogInternal is a helper for logging internal messages using an existing LoggerEntry.
func (le *LoggerEntry) LogInternal(level, process, message string) {
	le.ProcessName = process
	le.LoggingData(level, nil, ResponseLog{
		Status:  level,
		Message: message,
	})
}

// ensureElasticsearchClient initializes EsClient jika belum ada (thread-safe)
func ensureElasticsearchClient() error {
	esClientMutex.RLock()
	if EsClient != nil {
		esClientMutex.RUnlock()
		return nil
	}
	esClientMutex.RUnlock()

	esClientMutex.Lock()
	defer esClientMutex.Unlock()

	// Double-check after acquiring write lock
	if EsClient != nil {
		return nil
	}

	client, err := InitElasticsearch()
	if err != nil {
		return err
	}

	EsClient = client
	return nil
}

// InitElasticsearch initializes and tests the Elasticsearch client
func InitElasticsearch() (*elasticsearch.Client, error) {
	if appConfig == nil {
		return nil, fmt.Errorf("application config is nil")
	}

	esConfig := elasticsearch.Config{
		Addresses: []string{appConfig.ElasticSearch.ElasticHost},
		Username:  appConfig.ElasticSearch.ElasticUsername,
		Password:  appConfig.ElasticSearch.ElasticPassword,
		APIKey:    appConfig.ElasticSearch.ElasticApiKey,
	}

	esClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	res, err := esClient.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get Elasticsearch info: %w", err)
	}

	defer res.Body.Close()

	return esClient, nil
}
