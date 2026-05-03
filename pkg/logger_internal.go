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
	"github.com/natefinch/lumberjack"
)

// Global logging state
var (
	// EsClient is protected by esClientMutex for thread safety.
	EsClient      *elasticsearch.Client
	esClientMutex sync.RWMutex

	logFileWriter *lumberjack.Logger
	onceLogger    sync.Once

	logChan    chan logRequest
	onceWorker sync.Once

	cachedWd     string
	onceCachedWd sync.Once

	onceDirChecked sync.Once
)

// logRequest represents an internal request to process a log entry.
type logRequest struct {
	entry   *LoggerEntry
	payload any // Payload for Elasticsearch processing
	mode    string
}

// initLogWorker initializes the background log processing goroutine exactly once.
func initLogWorker() {
	onceWorker.Do(func() {
		logChan = make(chan logRequest, 1000)
		go func() {
			for req := range logChan {
				processLogRequest(req)
			}
		}()
	})
}

// processLogRequest dispatches a log request to the appropriate output destination(s).
func processLogRequest(req logRequest) {
	switch req.mode {
	case "console":
		req.entry.writeToConsole()
	case "file":
		req.entry.writeToLogFile()
	case "elasticsearch":
		req.entry.writeToElasticsearch(req.payload)
	case "hybrid":
		req.entry.writeToConsole()
		req.entry.writeToLogFile()
		req.entry.writeToElasticsearch(req.payload)
	}
}

// initLoggerFile sets up the rolling file writer exactly once.
func initLoggerFile() {
	onceLogger.Do(func() {
		onceDirChecked.Do(func() {
			_ = ensureLogDirectoryExists()
		})

		logFileName := generateLogFileName()
		logFilePath := path.Join(LogDir, logFileName)

		logFileWriter = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   false,
		}
	})
}

// writeToLogFile serializes the entry to JSON and appends it to the rolling log file.
func (le *LoggerEntry) writeToLogFile() {
	logMessage, _ := json.Marshal(le)
	if logFileWriter != nil {
		_, _ = fmt.Fprintf(logFileWriter, "%s\n", string(logMessage))
	}
}

// writeToConsole prints the entry as JSON to stdout.
func (le *LoggerEntry) writeToConsole() {
	data, _ := json.Marshal(le)
	fmt.Println(string(data))
}

// writeToElasticsearch sends the entry and its payload to an Elasticsearch index.
// The index name follows the pattern: <ELASTIC_PREFIX_INDEX><YYYYMMDD>
func (le *LoggerEntry) writeToElasticsearch(payload any) {
	if err := ensureElasticsearchClient(); err != nil {
		return
	}

	conf := getAppConfig()
	if conf == nil {
		return
	}

	payloadMap, _ := processPayload(payload)
	le.Request.Payload = payloadMap
	data, _ := json.Marshal(le)

	esIndexWithDate := conf.ElasticSearch.ElasticPrefixIndex + time.Now().Format("20060102")
	res, err := EsClient.Index(esIndexWithDate, bytes.NewReader(data))
	if err == nil {
		_ = res.Body.Close()
	}
}

// processPayload normalizes various payload types into a map[string]any for JSON logging.
func processPayload(payload any) (map[string]any, error) {
	if payload == nil {
		return nil, nil
	}

	switch v := payload.(type) {
	case map[string]any:
		return v, nil

	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return parsed, nil
		}
		return map[string]any{"raw": v}, nil

	case []byte:
		var parsed map[string]any
		if err := json.Unmarshal(v, &parsed); err == nil {
			return parsed, nil
		}
		return map[string]any{"raw": string(v)}, nil

	default:
		dataBytes, err := json.Marshal(v)
		if err != nil {
			return map[string]any{"error": "failed_to_marshal", "raw": fmt.Sprintf("%v", v)}, nil
		}
		var payloadMap map[string]any
		if err := json.Unmarshal(dataBytes, &payloadMap); err != nil {
			return map[string]any{"error": "failed_to_unmarshal", "raw": string(dataBytes)}, nil
		}
		return payloadMap, nil
	}
}

// generateLogFileName creates a daily log file name with an optional prefix from config.
func generateLogFileName() string {
	now := time.Now()
	prefix := "orion-to-core"
	if conf := getAppConfig(); conf != nil && conf.App.LogFilePrefix != "" {
		prefix = conf.App.LogFilePrefix
	}
	return fmt.Sprintf("%s-%d-%02d-%02d.log", prefix, now.Year(), int(now.Month()), now.Day())
}

// ensureLogDirectoryExists creates the log directory if it doesn't already exist.
func ensureLogDirectoryExists() error {
	return os.MkdirAll(LogDir, 0755)
}

// ensureElasticsearchClient initializes the ES client if it hasn't been initialized yet.
func ensureElasticsearchClient() error {
	esClientMutex.Lock()
	defer esClientMutex.Unlock()
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
