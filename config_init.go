package nanopony

import (
	"os"
	"slices"
	"strings"
)

// envConfig defines validation rules for an environment variable
type envConfig struct {
	name        string   // Environment variable name
	validValues []string // Allowed values (empty means any value is valid)
	defaultVal  string   // Default value if validation fails
}

// oracleEnv holds the environment variable names for Oracle connection
type oracleEnv struct {
	host     string
	port     string
	database string
	username string
	password string
}

// Environment variable configurations for application settings
var (
	// appEnvConfig defines validation for GO_ENV
	appEnvConfig = envConfig{
		name:        "GO_ENV",
		validValues: []string{"localhost", "staging", "production"},
		defaultVal:  "local",
	}

	// kafkaEnvConfig defines validation for KAFKA_MODELS
	kafkaEnvConfig = envConfig{
		name:        "KAFKA_MODELS",
		validValues: []string{"kafka-staging", "kafka-production", "kafka-confluent"},
		defaultVal:  "kafka-localhost",
	}

	// operationEnvConfig defines validation for OPERATION (any value allowed)
	operationEnvConfig = envConfig{
		name:        "OPERATION",
		validValues: []string{},
		defaultVal:  "default",
	}
)

// getEnvValue retrieves an environment variable value with validation.
// If the value is not in validValues list, it returns the default value.
// If validValues is empty, any value is accepted.
func getEnvValue(cfg envConfig) string {
	value := strings.ToLower(os.Getenv(cfg.name))

	// If value doesn't exist, return default
	if value == "" {
		return cfg.defaultVal
	}

	// If no validation rules, return value as-is
	if len(cfg.validValues) == 0 {
		return value
	}

	// Return value if it's valid
	if slices.Contains(cfg.validValues, value) {
		return value
	}

	// Return default if validation fails
	return cfg.defaultVal
}

// getOracleEnv returns Oracle environment variable names based on the environment.
// For "staging" environment, it returns staging variable names.
// For all other environments, it returns production variable names.
//
// Example:
//
//	env := getOracleEnv("staging")
//	// Returns: {host: "HOST_STAGING", port: "PORT_STAGING", ...}
func getOracleEnv(env string) oracleEnv {
	switch env {
	case "staging":
		return oracleEnv{
			host:     "HOST_STAGING",
			port:     "PORT_STAGING",
			database: "DATABASE_STAGING",
			username: "USERNAME_STAGING",
			password: "PASSWORD_STAGING",
		}
	case "localhost":
		return oracleEnv{
			host:     "HOST_STAGING",
			port:     "PORT_STAGING",
			database: "DATABASE_STAGING",
			username: "USERNAME_STAGING",
			password: "PASSWORD_STAGING",
		}
	}
	return oracleEnv{
		host:     "HOST_PRODUCTION",
		port:     "PORT_PRODUCTION",
		database: "DATABASE_PRODUCTION",
		username: "USERNAME_PRODUCTION",
		password: "PASSWORD_PRODUCTION",
	}
}

// getKafkaBrokers retrieves Kafka broker addresses from environment variables.
// The broker variable name is determined by the environment and Kafka model.
//
// Mapping:
//   - kafka-production + production env -> KAFKA_BROKERS_PRODUCTION
//   - kafka-staging (any env) -> KAFKA_BROKERS_STAGING
//   - Other combinations -> empty slice
func getKafkaBrokers(conf *Config) []string {
	env := conf.App.Env
	kafkaModel := initKafkaModels(conf)

	var brokerEnv string
	switch {
	case kafkaModel == "kafka-production" && env == "production":
		brokerEnv = "KAFKA_BROKERS_PRODUCTION"
	case kafkaModel == "kafka-staging" && env == "staging":
		brokerEnv = "KAFKA_BROKERS_STAGING"
	case kafkaModel == "kafka-staging" && env == "localhost":
		brokerEnv = "KAFKA_BROKERS_STAGING"
	default:
		return []string{}
	}

	if broker, exists := os.LookupEnv(brokerEnv); exists && broker != "" {
		return strings.Split(broker, ",")
	}

	return []string{}
}

// initKafkaModels initializes Kafka_models configuration (KAFKA_MODELS)
func initKafkaModels(conf *Config) string {
	conf.App.KafkaModels = getEnvValue(kafkaEnvConfig)
	return conf.App.KafkaModels
}

// initApp initializes application-level configuration (GO_ENV)
func initApp(conf *Config) {
	conf.App.Env = getEnvValue(appEnvConfig)
}

// initKafka initializes Kafka configuration.
// If the model is kafka-confluent, it delegates to initKafkaConfluent.
// Otherwise, it loads broker addresses from environment.
func initKafka(conf *Config) {
	if conf.App.KafkaModels == "kafka-confluent" {
		initKafkaConfluent(conf)
		return
	}
	conf.Kafka.Brokers = getKafkaBrokers(conf)
}

// initOracle initializes Oracle database configuration from environment variables.
// The variable names depend on the environment (staging vs production).
func initOracle(conf *Config) {
	env := getOracleEnv(conf.App.Env)
	conf.Oracle.Username = os.Getenv(env.username)
	conf.Oracle.Password = os.Getenv(env.password)
	conf.Oracle.Host = os.Getenv(env.host)
	conf.Oracle.Port = os.Getenv(env.port)
	conf.Oracle.DatabaseName = os.Getenv(env.database)
}

// initOperation initializes operation configuration
func initOperation(conf *Config) {
	conf.App.Operation = getEnvValue(operationEnvConfig)
}

// initKafkaConfluent initializes Confluent Cloud Kafka configuration.
// It loads API key, secret, resource ID, and bootstrap server from environment.
func initKafkaConfluent(conf *Config) {
	conf.KafkaConfluent.ApiKey = os.Getenv("API_KEY_KAFKA_CONFLUENT")
	conf.KafkaConfluent.ApiSecret = os.Getenv("API_SECRET_KAFKA_CONFLUENT")
	conf.KafkaConfluent.Resource = os.Getenv("RESOURCE_KAFKA_CONFLUENT")

	if bootstrapServer, exists := os.LookupEnv("BOOTSTRAP_SERVER_KAFKA_CONFLUENT"); exists && bootstrapServer != "" {
		conf.KafkaConfluent.BootstrapServers = []string{bootstrapServer}
	}
}

// initElasticSearch initializes Elasticsearch configuration from environment variables
func initElasticSearch(conf *Config) {
	conf.ElasticSearch.ElasticHost = os.Getenv("ELASTIC_HOST")
	conf.ElasticSearch.ElasticPassword = os.Getenv("ELASTIC_PASSWORD")
	conf.ElasticSearch.ElasticUsername = os.Getenv("ELASTIC_USERNAME")
	conf.ElasticSearch.ElasticIndexName = os.Getenv("ELASTIC_INDEX_DATA")
	conf.ElasticSearch.ElasticApiKey = os.Getenv("ELASTIC_API_KEY")
	conf.ElasticSearch.ElasticPrefixIndex = os.Getenv("ELASTIC_PREFIX_INDEX")
}

// WithOperationValidValues sets valid values for the OPERATION environment variable.
// This is useful for defining custom operation modes.
//
// Example:
//
//	WithOperationValidValues([]string{"read-only", "write", "admin"})
func WithOperationValidValues(values []string) {
	operationEnvConfig.validValues = values
}

// getEnvByPrefix retrieves all environment variables with a given prefix.
// This is used by LoadDynamic to dynamically load configuration.
//
// Example:
//
//	// Get all CUSTOM_* environment variables
//	vars := getEnvByPrefix("CUSTOM_")
//	// Returns map with keys like "CUSTOM_API_URL", "CUSTOM_TIMEOUT", etc.
func getEnvByPrefix(prefix string) map[string]string {
	result := make(map[string]string)

	// Get all environment variables
	for _, env := range os.Environ() {
		// Split on first "=" to get key and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// If prefix is empty, load everything
		// If prefix is set, only load vars with that prefix
		if prefix == "" || strings.HasPrefix(key, prefix) {
			result[key] = value
		}
	}

	return result
}
