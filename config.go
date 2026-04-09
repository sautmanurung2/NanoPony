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

import "github.com/joho/godotenv"

// Config holds all configuration for the framework.
// This struct contains all subsystem configurations:
// App, Oracle, Kafka, Confluent Cloud, and Elasticsearch.
//
// Configuration is loaded from environment variables automatically.
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
var appConfig *Config

// NewConfig initializes and returns a new Config instance.
// It loads environment variables from .env file if present.
// Configuration is loaded once and cached (singleton pattern).
//
// Environment Variables Required:
//   - GO_ENV: local, staging, or production
//   - KAFKA-MODELS: kafka-localhost, kafka-staging, kafka-production, or kafka-confluent
//
// Example:
//
//	config := nanopony.NewConfig()
func NewConfig() *Config {
	_ = godotenv.Load()
	if appConfig == nil {
		appConfig = &Config{}
		initApp(appConfig)
		initKafka(appConfig)
		initOracle(appConfig)
		initOperation(appConfig)
		initElasticSearch(appConfig)
	}
	return appConfig
}

// ResetConfig resets the configuration singleton.
// This is primarily useful for testing purposes.
//
// Example:
//
//	defer nanopony.ResetConfig() // Reset after test
func ResetConfig() {
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
	conf := &Config{}

	initApp(conf)
	initKafka(conf)
	initOracle(conf)
	initOperation(conf)
	initElasticSearch(conf)

	for _, fn := range initFuncs {
		fn(conf)
	}

	return conf
}
