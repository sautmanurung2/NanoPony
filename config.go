// Package nanopony provides a Kafka-Oracle integration framework
// with worker pool and polling capabilities.
//
// Example usage:
//
//	config := nanopony.NewConfig()
//	framework := nanopony.NewFramework().
//		WithConfig(config).
//		WithDatabase().
//		WithKafkaWriter().
//		WithProducer().
//		WithWorkerPool(5, 100)
//	components := framework.Build()
package nanopony

import (
	"fmt"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the framework.
// This struct contains all subsystem configurations:
// App, Oracle, Kafka, Confluent Cloud, and Elasticsearch.
//
// Configuration is loaded from environment variables automatically.
// The Dynamic field allows loading arbitrary environment variables without
// modifying the framework code.
type Config struct {
	// App holds application-level configuration (environment, Kafka model, operation mode)
	App AppConfig
	// Oracle holds Oracle database connection details
	Oracle OracleConfig
	// Kafka holds standard Kafka broker configuration
	Kafka KafkaConfig
	// KafkaConfluent holds Confluent Cloud specific configuration
	KafkaConfluent KafkaConfluentConfig
	// ElasticSearch holds Elasticsearch connection details
	ElasticSearch ElasticSearchConfig
	// Dynamic is a map for arbitrary environment variables.
	// This allows adding new env vars without modifying the framework.
	// Use prefixes like "CUSTOM_" to group related variables.
	Dynamic map[string]string
}

// AppConfig holds application-level configuration
type AppConfig struct {
	// Env is the environment: local, staging, production
	Env string
	// KafkaModels specifies which Kafka configuration to use:
	// kafka-localhost, kafka-staging, kafka-production, kafka-confluent
	KafkaModels string
	// Operation is the operation mode (custom, can be defined per application)
	Operation string
}

// OracleConfig holds Oracle database configuration
type OracleConfig struct {
	Username     string
	Password     string
	Host         string
	Port         string
	DatabaseName string
}

// ElasticSearchConfig holds Elasticsearch connection configuration
type ElasticSearchConfig struct {
	ElasticHost        string
	ElasticPassword    string
	ElasticUsername    string
	ElasticIndexName   string
	ElasticApiKey      string
	ElasticPrefixIndex string
}

// KafkaConfig holds standard Kafka broker configuration
type KafkaConfig struct {
	Brokers []string
}

// KafkaConfluentConfig holds Confluent Cloud specific Kafka configuration
type KafkaConfluentConfig struct {
	ApiKey           string
	ApiSecret        string
	Resource         string
	BootstrapServers []string
}

// appConfig is the global configuration singleton.
// Use NewConfig() or BuildConfig() to initialize it.
// Protected by configMutex for thread safety.
var appConfig *Config
var configMutex sync.RWMutex

// NewConfig initializes and returns a new Config instance.
// It loads environment variables from .env file if present.
// Configuration is loaded once and cached (singleton pattern).
//
// Environment Variables Required:
//   - GO_ENV: local, staging, or production
//   - KAFKA_MODELS: kafka-localhost, kafka-staging, kafka-production, or kafka-confluent
//
// Example:
//
//	config := nanopony.NewConfig()
func NewConfig() *Config {
	_ = godotenv.Load()

	configMutex.RLock()
	if appConfig != nil {
		configMutex.RUnlock()
		return appConfig
	}
	configMutex.RUnlock()

	configMutex.Lock()
	defer configMutex.Unlock()

	// Double-check after acquiring write lock
	if appConfig != nil {
		return appConfig
	}

	appConfig = &Config{
		Dynamic: make(map[string]string),
	}
	initApp(appConfig)
	initKafkaModels(appConfig)
	initKafka(appConfig)
	initOracle(appConfig)
	initOperation(appConfig)
	initElasticSearch(appConfig)

	return appConfig
}

// Validate checks if the configuration is valid and all required fields are set.
// This is used for fail-fast behavior during framework initialization.
func (c *Config) Validate() error {
	if c.App.Env == "" {
		return fmt.Errorf("GO_ENV is not set (local, staging, or production)")
	}
	if c.App.KafkaModels == "" {
		return fmt.Errorf("KAFKA_MODELS is not set")
	}

	// Validate Kafka based on chosen model
	switch c.App.KafkaModels {
	case "kafka-confluent":
		if len(c.KafkaConfluent.BootstrapServers) == 0 {
			return fmt.Errorf("KAFKA_CONFLUENT bootstrap servers are not set")
		}
		if c.KafkaConfluent.ApiKey == "" || c.KafkaConfluent.ApiSecret == "" {
			return fmt.Errorf("KAFKA_CONFLUENT credentials (API Key/Secret) are not set")
		}
	case "kafka-staging", "kafka-production":
		// For staging/production, brokers must be set
		if len(c.Kafka.Brokers) == 0 {
			return fmt.Errorf("KAFKA brokers are not set for model %s", c.App.KafkaModels)
		}
	}

	// Oracle validation is handled by NewOracleFromConfig when WithDatabase() is called.
	// We only check if the model is set for now.

	return nil
}

// ResetConfig resets the configuration singleton.
// This is primarily useful for testing purposes.
//
// Example:
//
//	defer nanopony.ResetConfig() // Reset after test
func ResetConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()
	appConfig = nil
}

// BuildConfig builds configuration with custom initializers.
// This allows for custom configuration logic beyond the default initializers.
//
// Example:
//
//	config := BuildConfig(func(c *Config) {
//	    c.App.Env = "custom"
//	})
func BuildConfig(initFuncs ...func(*Config)) *Config {
	_ = godotenv.Load()
	conf := &Config{
		Dynamic: make(map[string]string),
	}

	initApp(conf)
	initKafkaModels(conf)
	initKafka(conf)
	initOracle(conf)
	initOperation(conf)
	initElasticSearch(conf)

	for _, fn := range initFuncs {
		fn(conf)
	}

	configMutex.Lock()
	appConfig = conf
	configMutex.Unlock()

	return conf
}

// LoadDynamic loads environment variables with the given prefix into the Dynamic map.
// If prefix is empty, it loads ALL environment variables (use with caution).
// This allows dynamic configuration without modifying the framework code.
//
// Example:
//
//	// Load all CUSTOM_* environment variables
//	config.LoadDynamic("CUSTOM_")
//
//	// This will load CUSTOM_API_URL, CUSTOM_TIMEOUT, etc.
func (c *Config) LoadDynamic(prefix string) {
	if c.Dynamic == nil {
		c.Dynamic = make(map[string]string)
	}
	envVars := getEnvByPrefix(prefix)
	for key, value := range envVars {
		c.Dynamic[key] = value
	}
}
