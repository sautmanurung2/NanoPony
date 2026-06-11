package nanopony

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/natefinch/lumberjack"
)

// logRequest represents an internal request to process a log entry.
type logRequest struct {
	entry *LoggerEntry
	mode  string
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
		manager:         le.manager,
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

// initWorkers initializes the background log processing goroutines for a LogManager instance.
func (lm *LogManager) initWorkers() {
	// 1. Main Dispatcher Worker
	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()
		for req := range lm.logChan {
			lm.dispatchLogRequest(req)
		}
		// When logChan is closed, close all sink channels
		close(lm.consoleChan)
		close(lm.fileChan)
		close(lm.esChan)
	}()

	// 2. Dedicated Console Worker
	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()
		for req := range lm.consoleChan {
			req.entry.writeToConsole()
		}
	}()

	// 3. Dedicated File Worker
	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()
		for req := range lm.fileChan {
			req.entry.writeToLogFile(lm)
		}
	}()

	// 4. Dedicated Elasticsearch Worker
	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()
		for req := range lm.esChan {
			req.entry.writeToElasticsearch(lm)
		}
	}()
}

// dispatchLogRequest routes a log request to the appropriate sink workers.
func (lm *LogManager) dispatchLogRequest(req logRequest) {
	switch req.mode {
	case "console":
		lm.sendToSink(lm.consoleChan, req)
	case "file":
		lm.sendToSink(lm.fileChan, req)
	case "elasticsearch":
		lm.sendToSink(lm.esChan, req)
	case "hybrid":
		ce1 := req.entry.clone()
		ce2 := req.entry.clone()

		lm.sendToSink(lm.consoleChan, logRequest{entry: ce1, mode: req.mode})
		lm.sendToSink(lm.fileChan, logRequest{entry: ce2, mode: req.mode})
		lm.sendToSink(lm.esChan, logRequest{entry: req.entry, mode: req.mode})
	}
}

// sendToSink attempts to send a request to a sink channel without blocking the dispatcher.
func (lm *LogManager) sendToSink(ch chan logRequest, req logRequest) {
	if ch == nil {
		return
	}
	select {
	case ch <- req:
	case <-lm.ctx.Done():
		return
	default:
		// Drop if sink is full
	}
}

// writeToLogFile serializes the entry to JSON and appends it to the rolling log file.
func (le *LoggerEntry) writeToLogFile(lm *LogManager) {
	if lm.logFileWriter == nil {
		lm.initLoggerFile()
	}

	le.mu.RLock()
	logMessage, err := json.Marshal(le)
	le.mu.RUnlock()

	if err != nil {
		return
	}

	if lm.logFileWriter != nil {
		_, _ = fmt.Fprintf(lm.logFileWriter, "%s\n", string(logMessage))
	}
}

// initLoggerFile sets up the rolling file writer for a LogManager.
func (lm *LogManager) initLoggerFile() {
	_ = os.MkdirAll(LogDir, 0755)

	logFileName := lm.generateLogFileName()
	logFilePath := path.Join(LogDir, logFileName)

	lm.logFileWriter = &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}
}

// writeToConsole prints the entry as JSON to stdout.
func (le *LoggerEntry) writeToConsole() {
	le.mu.RLock()
	data, err := json.Marshal(le)
	le.mu.RUnlock()

	if err == nil {
		fmt.Println(string(data))
	}
}

// writeToElasticsearch sends the entry and its payload to an Elasticsearch index.
func (le *LoggerEntry) writeToElasticsearch(lm *LogManager) {
	if lm.esClient == nil {
		if err := lm.initElasticsearchClient(); err != nil {
			return
		}
	}

	le.mu.RLock()
	data, err := json.Marshal(le)
	le.mu.RUnlock()

	if err != nil {
		return
	}

	esIndexWithDate := lm.config.ElasticSearch.ElasticPrefixIndex + time.Now().Format("20060102")
	res, err := lm.esClient.Index(esIndexWithDate, bytes.NewReader(data))
	if err == nil {
		_ = res.Body.Close()
	}
}

// initElasticsearchClient initializes the ES client for a LogManager.
func (lm *LogManager) initElasticsearchClient() error {
	if lm.config == nil {
		return fmt.Errorf("config not set")
	}

	es := lm.config.EnsureElasticSearch()
	cfg := elasticsearch.Config{
		Addresses: []string{es.ElasticHost},
		Username:  es.ElasticUsername,
		Password:  es.ElasticPassword,
		APIKey:    es.ElasticApiKey,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}
	lm.esClient = client
	return nil
}

// processPayload normalizes various payload types into a map[string]any for JSON logging.
func processPayload(payload any) map[string]any {
	if payload == nil {
		return nil
	}

	switch p := payload.(type) {
	case map[string]any:
		return deepCopyMap(p, 0)
	case string:
		var m map[string]any
		if err := json.Unmarshal([]byte(p), &m); err != nil {
			return map[string]any{"raw": p}
		}
		return m
	case []byte:
		var m map[string]any
		if err := json.Unmarshal(p, &m); err != nil {
			return map[string]any{"raw": string(p)}
		}
		return m
	default:
		dataBytes, err := json.Marshal(payload)
		if err != nil {
			return map[string]any{"error": "failed_to_marshal", "raw": fmt.Sprintf("%v", payload)}
		}

		var payloadMap map[string]any
		if err := json.Unmarshal(dataBytes, &payloadMap); err != nil {
			return map[string]any{"raw": string(dataBytes)}
		}
		return payloadMap
	}
}

const maxCopyDepth = 10

// deepCopyMap performs a shallow copy of a map, and recursively copies nested maps.
func deepCopyMap(m map[string]any, depth int) map[string]any {
	if depth > maxCopyDepth {
		return map[string]any{"error": "max_copy_depth_reached"}
	}

	cp := make(map[string]any, len(m))
	for k, v := range m {
		if vm, ok := v.(map[string]any); ok {
			cp[k] = deepCopyMap(vm, depth+1)
		} else {
			cp[k] = v
		}
	}
	return cp
}

// generateLogFileName creates a daily log file name with an optional prefix from config.
func (lm *LogManager) generateLogFileName() string {
	now := time.Now()
	prefix := "nanopony"
	if lm.config != nil && lm.config.App.LogFilePrefix != "" {
		prefix = lm.config.App.LogFilePrefix
	}
	return fmt.Sprintf("%s-%d-%02d-%02d.log", prefix, now.Year(), int(now.Month()), now.Day())
}
