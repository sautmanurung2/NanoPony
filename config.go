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

// Config holds all configuration for the framework
type Config struct {
	App            AppConfig
	Oracle         OracleConfig
	Kafka          KafkaConfig
	KafkaConfluent KafkaConfluentConfig
	ElasticSearch  ElasticSearchConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	// Env is the environment: local, staging, production
	Env string
	// KafkaModels specifies which Kafka configuration to use:
	// kafka-localhost, kafka-staging, kafka-production, kafka-confluent
	KafkaModels string
	// Operation is the operation mode
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

type ElasticSearchConfig struct {
	ElasticHost        string
	ElasticPassword    string
	ElasticUsername    string
	ElasticIndexName   string
	ElasticApiKey      string
	ElasticPrefixIndex string
}

// KafkaConfig holds standard Kafka configuration
type KafkaConfig struct {
	Brokers []string
}

// KafkaConfluentConfig holds Confluent Cloud Kafka configuration
type KafkaConfluentConfig struct {
	ApiKey           string
	ApiSecret        string
	Resource         string
	BootstrapServers []string
}

var appConfig *Config

// NewConfig initializes and returns a new Config instance.
// It loads environment variables from .env file if present.
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
// Useful for testing purposes.
func ResetConfig() {
	appConfig = nil
}

// BuildConfig builds configuration with custom initializers.
// This allows for custom configuration logic beyond the default initializers.
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
