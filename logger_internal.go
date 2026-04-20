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
	payload any     // Still needed for ES processing
	mode    string
}

// initLogWorker initializes the background log processing goroutine.
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

// processLogRequest dispatches a log request to the appropriate destination.
func processLogRequest(req logRequest) {
	switch req.mode {
	case "console":
		req.entry.printToConsole()
	case "file":
		req.entry.sendToFileNow()
	case "elasticsearch":
		req.entry.sendToElasticSearchNow(req.payload)
	case "hybrid":
		req.entry.printToConsole()
		req.entry.sendToFileNow()
		req.entry.sendToElasticSearchNow(req.payload)
	}
}

// initLoggerFile sets up the rolling file writer.
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

// Helper methods for internal logging machinery

func (le *LoggerEntry) sendToFileNow() {
	logMessage, _ := json.Marshal(le)
	if logFileWriter != nil {
		_, _ = fmt.Fprintf(logFileWriter, "%s\n", string(logMessage))
	}
}

func (le *LoggerEntry) printToConsole() {
	data, _ := json.Marshal(le)
	fmt.Println(string(data))
}

func (le *LoggerEntry) sendToElasticSearchNow(payload any) {
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

func processPayload(payload any) (map[string]any, error) {
	var payloadMap map[string]any

	switch v := payload.(type) {
	case map[string]any:
		payloadMap = v

	case string:
		var parsed map[string]any
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			payloadMap = parsed
		} else {
			payloadMap = map[string]any{
				"raw": v,
			}
		}
	case []byte:
		var parsed map[string]any
		if err := json.Unmarshal(v, &parsed); err == nil {
			payloadMap = parsed
		} else {
			payloadMap = map[string]any{
				"raw": string(v),
			}
		}

	default:
		if v == nil {
			return nil, nil
		}
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

func generateLogFileName() string {
	now := time.Now()
	prefix := "orion-to-core"
	conf := getAppConfig()
	if conf != nil && conf.App.LogFilePrefix != "" {
		prefix = conf.App.LogFilePrefix
	}
	return fmt.Sprintf("%s-%d-%02d-%02d.log", prefix, now.Year(), int(now.Month()), now.Day())
}

func ensureLogDirectoryExists() error {
	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		return os.Mkdir(LogDir, 0755)
	}
	return nil
}

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
