package nanopony

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
	"github.com/natefinch/lumberjack"
)

var (
	EsClient *elasticsearch.Client
	_        = godotenv.Load()

	logFileWriter *lumberjack.Logger
	onceLogger    sync.Once
)

func initLoggerFile() {
	onceLogger.Do(func() {
		ensureLogDirectoryExists()

		logFileName := generateLogFileName()
		logFilePath := path.Join("./src/logs", logFileName)

		logFileWriter = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   false,
		}
	})
}

type LoggerEntry struct {
	StartTimestamp  time.Time `json:"start_timestamp"`
	EndTimestamp    time.Time `json:"end_timestamp"`
	ReferenceId     string    `json:"reference_id"`
	ReferenceNumber string    `json:"reference_number"`
	ProcessName     string    `json:"process_name"`
	SystemName      string    `json:"system_name"`
	Entity          string    `json:"entity"`
	Additionals     string    `json:"additionals"`
	Duration        int64     `json:"duration"`
	Service         string    `json:"service"`
	Path            string    `json:"path"`
	Level           string    `json:"level"`
	UserLogin       string    `json:"user_login"`
	NodeCode        string    `json:"node_code"`
	Request         struct {
		Payload map[string]any `json:"payload"`
	} `json:"request"`
	Response Response `json:"response"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

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
		Path:            dir,
		NodeCode:        "",
	}

	return loggerEntry
}

func (le *LoggerEntry) SendToFile(level string, response Response) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response
	logMessage, _ := json.Marshal(le)

	_, err := fmt.Fprintf(logFileWriter, "%s\n", string(logMessage))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
	}
}

func (le *LoggerEntry) LoggingData(
	level string,
	payload any,
	response Response,
) {
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response

	payloadMap, err := processPayload(payload)
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error processing payload: %v\n", err)
		return
	}

	le.Request.Payload = payloadMap
	outputMode := os.Getenv("LOG_OUTPUT_MODE")
	if outputMode == "" {
		outputMode = "hybrid"
	}

	switch outputMode {
	case "fluentd":
		le.printToConsole()
	case "elasticsearch":
		le.SendToElasticSearch(level, payload, response)
	case "hybrid":
		le.printToConsole()
		le.SendToElasticSearch(level, payload, response)
	default:
		fmt.Printf("[Warning] Unknown LOG_OUTPUT_MODE: %s. Defaulting to console.\n", outputMode)
		le.printToConsole()
	}
}

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

func generateLogFileName() string {
	currentTime := time.Now()
	year := currentTime.Year()
	month := int(currentTime.Month())
	day := currentTime.Day()

	return fmt.Sprintf("orion-to-core-%d-%02d-%02d.log", year, month, day)
}

func ensureLogDirectoryExists() {
	logFilePath := "./logs"
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		err := os.Mkdir(logFilePath, os.ModePerm)
		if err != nil {
			fmt.Printf("Failed to create log directory: %s\n", err)
			os.Exit(1)
		}
	}
}

func (le *LoggerEntry) printToConsole() {
	data, err := json.Marshal(le)
	if err != nil {
		fmt.Printf("Error marshaling for console: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

func (le *LoggerEntry) SendToElasticSearch(
	level string,
	payload any,
	response Response,
) {
	if EsClient == nil {
		client, err := le.InitElasticSearch()
		if err != nil {
			_, _ = fmt.Fprintf(logFileWriter, "Failed to initialize Elasticsearch client: %v\n", err)
			return
		}
		EsClient = client
	}

	esClient := EsClient
	le.EndTimestamp = time.Now()
	le.Duration = le.EndTimestamp.Sub(le.StartTimestamp).Milliseconds()
	le.Level = level
	le.Response = response

	payloadMap, err := processPayload(payload)
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error processing payload: %v\n", err)
		return
	}

	le.Request.Payload = payloadMap
	data, err := json.Marshal(le)
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error marshaling message to JSON: %v\n", err)
		return
	}

	esIndexWithDate := appConfig.ElasticSearch.ElasticPrefixIndex + time.Now().Format("20060102")
	esResult, err := esClient.Index(esIndexWithDate, bytes.NewReader(data))
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error indexing message to Elasticsearch: %v\n", err)
		return
	}
	defer esResult.Body.Close()

	body, err := io.ReadAll(esResult.Body)
	if err != nil {
		fmt.Println("Failed to read Elasticsearch response body:", err)
		return
	}

	var esResponse map[string]any
	if err := json.Unmarshal(body, &esResponse); err != nil {
		fmt.Println("Failed to parse Elasticsearch response:", err)
		return
	}

	if errorMsg, hasError := esResponse["error"]; hasError {
		fmt.Printf("error send to elastic: ❌ Document rejected by Elasticsearch: %s\n", errorMsg)
		_, _ = fmt.Fprintf(logFileWriter, "Elasticsearch rejected document: %v\n", errorMsg)
		return
	}

	_, _ = fmt.Fprintf(logFileWriter, "%s\n", string(data))
}

func (le *LoggerEntry) InitElasticSearch() (*elasticsearch.Client, error) {
	esConfig := elasticsearch.Config{
		Addresses: []string{appConfig.ElasticSearch.ElasticHost},
		Username:  appConfig.ElasticSearch.ElasticUsername,
		Password:  appConfig.ElasticSearch.ElasticPassword,
		APIKey:    appConfig.ElasticSearch.ElasticApiKey,
	}

	esClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error creating the client: %s\n", err)
	}

	res, err := esClient.Info()
	if err != nil {
		_, _ = fmt.Fprintf(logFileWriter, "Error getting Elasticsearch info: %v\n", err)
	}

	defer res.Body.Close()

	_, _ = fmt.Fprintf(logFileWriter, "Connected to Elasticsearch\n")

	return esClient, nil
}
