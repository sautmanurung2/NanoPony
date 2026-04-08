// Configuration: Application configuration management
package config

import (
	"fmt"
	"os"
	"time"
)

// AppConfig holds application-level configuration
type AppConfig struct {
	Env        string
	ServerPort string
	LogLevel   string
	Database   DatabaseConfig
	Kafka      KafkaConfig
	WorkerPool WorkerPoolConfig
	Poller     PollerConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	Database        string
	Username        string
	Password        string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnIdleTime    time.Duration
	ConnMaxLifetime time.Duration
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers           []string
	TopicPrefix       string
	UserEventsTopic   string
	OrderEventsTopic  string
	NotificationTopic string
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	NumWorkers int
	QueueSize  int
}

// PollerConfig holds poller configuration
type PollerConfig struct {
	Interval  time.Duration
	BatchSize int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *AppConfig {
	return &AppConfig{
		Env:        getEnv("GO_ENV", "local"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "1521"),
			Database:        getEnv("DB_NAME", "ORCL"),
			Username:        getEnv("DB_USERNAME", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnIdleTime:    5 * time.Minute,
			ConnMaxLifetime: 60 * time.Minute,
		},
		Kafka: KafkaConfig{
			Brokers:           getKafkaBrokers(),
			TopicPrefix:       getEnv("KAFKA_TOPIC_PREFIX", "dev"),
			UserEventsTopic:   getEnv("KAFKA_TOPIC_USER_EVENTS", "user-events"),
			OrderEventsTopic:  getEnv("KAFKA_TOPIC_ORDER_EVENTS", "order-events"),
			NotificationTopic: getEnv("KAFKA_TOPIC_NOTIFICATIONS", "notifications"),
		},
		WorkerPool: WorkerPoolConfig{
			NumWorkers: getEnvInt("WORKER_POOL_SIZE", 5),
			QueueSize:  getEnvInt("WORKER_QUEUE_SIZE", 100),
		},
		Poller: PollerConfig{
			Interval:  getEnvDuration("POLLER_INTERVAL", 1*time.Second),
			BatchSize: getEnvInt("POLLER_BATCH_SIZE", 100),
		},
	}
}

// KafkaTopic returns full Kafka topic name with prefix
func (c *AppConfig) KafkaTopic(baseTopic string) string {
	if c.Kafka.TopicPrefix == "" {
		return baseTopic
	}
	return fmt.Sprintf("%s.%s", c.Kafka.TopicPrefix, baseTopic)
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// getEnvInt retrieves environment variable as int or returns default
func getEnvInt(key string, defaultVal int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}

// getEnvDuration retrieves environment variable as duration or returns default
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultVal
	}
	return duration
}

// getKafkaBrokers retrieves Kafka brokers from environment
func getKafkaBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return []string{"localhost:9092"}
	}

	var result []string
	for _, broker := range splitString(brokers, ",") {
		if broker != "" {
			result = append(result, broker)
		}
	}
	return result
}

// splitString splits string by delimiter
func splitString(s, delimiter string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == delimiter {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
