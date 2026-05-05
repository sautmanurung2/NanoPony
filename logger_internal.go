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
	// esClient is protected by esClientMutex for thread safety.
	esClient      *elasticsearch.Client
	esClientMutex sync.RWMutex

	logFileWriter *lumberjack.Logger
	onceLogger    sync.Once

	logChan    chan logRequest
	consoleChan chan logRequest
	fileChan    chan logRequest
	esChan      chan logRequest
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

// clone creates a safe copy of the LoggerEntry metadata without copying the mutex.
func (le *LoggerEntry) clone() *LoggerEntry {
	le.mu.RLock()
	defer le.mu.RUnlock()

	return le.cloneUnlocked()
}

// cloneUnlocked creates a copy of the LoggerEntry without acquiring the lock.
// The caller MUST hold le.mu (either RLock or Lock) before calling.
func (le *LoggerEntry) cloneUnlocked() *LoggerEntry {
	return &LoggerEntry{
		StartTimestamp:  le.StartTimestamp,
		EndTimestamp:    le.EndTimestamp,
		ReferenceId:     le.ReferenceId,
		ReferenceNumber: le.ReferenceNumber,
		ProcessName:     le.ProcessName,
		SystemName:      le.SystemName,
		Entity:          le.Entity,
		Additionals:     le.Additionals,
		Duration:        le.Duration,
		Service:         le.Service,
		Path:            le.Path,
		Level:           le.Level,
		UserLogin:       le.UserLogin,
		NodeCode:        le.NodeCode,
		Request:         le.Request,
		Response:        le.Response,
	}
}


// initLogWorker initializes the background log processing goroutines exactly once.
func initLogWorker() {
	onceWorker.Do(func() {
		logChan = make(chan logRequest, 1000)
		consoleChan = make(chan logRequest, 1000)
		fileChan = make(chan logRequest, 1000)
		esChan = make(chan logRequest, 1000)

		// Main Dispatcher Worker
		go func() {
			for req := range logChan {
				dispatchLogRequest(req)
			}
		}()

		// Dedicated Console Worker (Realtime)
		go func() {
			for req := range consoleChan {
				req.entry.writeToConsole()
			}
		}()

		// Dedicated File Worker (Can be slow)
		go func() {
			for req := range fileChan {
				req.entry.writeToLogFile()
			}
		}()

		// Dedicated Elasticsearch Worker (Realtime/Network dependent)
		go func() {
			for req := range esChan {
				// Use the payload already inside req.entry
				req.entry.writeToElasticsearch(nil)
			}
		}()
	})
}

// dispatchLogRequest routes a log request to the appropriate sink workers.
func dispatchLogRequest(req logRequest) {
	switch req.mode {
	case "console":
		sendToSink(consoleChan, req)
	case "file":
		sendToSink(fileChan, req)
	case "elasticsearch":
		sendToSink(esChan, req)
	case "hybrid":
		// Create clones for parallel processing.
		// Each clone gets its own LoggerEntry metadata but shares the same 
		// processed (read-only) payload map.
		ce1 := req.entry.clone()
		ce2 := req.entry.clone()
		
		sendToSink(consoleChan, logRequest{entry: ce1, mode: req.mode})
		sendToSink(fileChan, logRequest{entry: ce2, mode: req.mode})
		sendToSink(esChan, logRequest{entry: req.entry, mode: req.mode})
	}
}

// sendToSink attempts to send a request to a sink channel without blocking the dispatcher.
func sendToSink(ch chan logRequest, req logRequest) {
	if ch == nil {
		return
	}
	select {
	case ch <- req:
	default:
		// If a specific sink is full, we drop the log for THAT sink only.
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
	le.mu.RLock()
	logMessage, _ := json.Marshal(le)
	le.mu.RUnlock()
	
	if logFileWriter != nil {
		_, _ = fmt.Fprintf(logFileWriter, "%s\n", string(logMessage))
	}
}

// writeToConsole prints the entry as JSON to stdout.
func (le *LoggerEntry) writeToConsole() {
	le.mu.RLock()
	data, _ := json.Marshal(le)
	le.mu.RUnlock()
	
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
	
	le.mu.Lock()
	le.Request.Payload = payloadMap
	data, _ := json.Marshal(le)
	le.mu.Unlock()

	esIndexWithDate := conf.ElasticSearch.ElasticPrefixIndex + time.Now().Format("20060102")
	res, err := esClient.Index(esIndexWithDate, bytes.NewReader(data))
	if err == nil {
		_ = res.Body.Close()
	}
}

// processPayload normalizes various payload types into a map[string]any for JSON logging.
func processPayload(payload any) (map[string]any, error) {
	if payload == nil {
		return nil, nil
	}

	// ALWAYS perform a deep copy for maps to avoid data races
	// when the original map is modified by the caller or other workers.
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{"error": "failed_to_marshal", "raw": fmt.Sprintf("%v", payload)}, nil
	}
	
	var payloadMap map[string]any
	if err := json.Unmarshal(dataBytes, &payloadMap); err != nil {
		// If it's a string or simple type, it might fail unmarshal to map
		return map[string]any{"raw": string(dataBytes)}, nil
	}
	return payloadMap, nil
}

// generateLogFileName creates a daily log file name with an optional prefix from config.
func generateLogFileName() string {
	now := time.Now()
	prefix := "nanopony"
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
	if esClient != nil {
		return nil
	}
	client, err := InitElasticsearch()
	if err != nil {
		return err
	}
	esClient = client
	return nil
}
