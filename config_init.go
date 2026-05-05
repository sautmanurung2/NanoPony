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
	// appEnvConfig defines validation for GO_ENV.
	// Valid values: localhost, staging, production. Default: local.
	appEnvConfig = envConfig{
		name:        "GO_ENV",
		validValues: []string{"localhost", "staging", "production"},
		defaultVal:  "local",
	}

	// kafkaEnvConfig defines validation for KAFKA_MODELS.
	// Valid values: kafka-staging, kafka-production, kafka-confluent. Default: kafka-localhost.
	kafkaEnvConfig = envConfig{
		name:        "KAFKA_MODELS",
		validValues: []string{"kafka-staging", "kafka-production", "kafka-confluent"},
		defaultVal:  "kafka-localhost",
	}

	// operationEnvConfig defines validation for OPERATION.
	// This is a custom field and accepts any value. Default: default.
	operationEnvConfig = envConfig{
		name:        "OPERATION",
		validValues: []string{},
		defaultVal:  "default",
	}

	// logFilePrefixEnvConfig defines the prefix for rolling log files.
	// Default: nanopony.
	logFilePrefixEnvConfig = envConfig{
		name:        "LOG_FILE_PREFIX",
		validValues: []string{},
		defaultVal:  "nanopony",
	}

	// logOutputModeEnvConfig defines where logs are sent.
	// Valid values: console, file, elasticsearch, hybrid. Default: hybrid.
	logOutputModeEnvConfig = envConfig{
		name:        "LOG_OUTPUT_MODE",
		validValues: []string{"console", "file", "elasticsearch", "hybrid"},
		defaultVal:  "hybrid",
	}
)

// getEnvValue retrieves an environment variable value with validation.
// It normalizes the value to lowercase and compares it against the validValues list.
// If the value is invalid or missing, it returns the default value.
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

// getOracleEnv returns Oracle environment variable names based on the environment string.
// It maps "staging" and "local" variants to staging environment variable names,
// and defaults to production environment variable names for everything else.
func getOracleEnv(env string) oracleEnv {
	switch env {
	case "staging", "localhost", "local":
		return oracleEnv{
			host:     "HOST_STAGING",
			port:     "PORT_STAGING",
			database: "DATABASE_STAGING",
			username: "USERNAME_STAGING",
			password: "PASSWORD_STAGING",
		}
	default:
		return oracleEnv{
			host:     "HOST_PRODUCTION",
			port:     "PORT_PRODUCTION",
			database: "DATABASE_PRODUCTION",
			username: "USERNAME_PRODUCTION",
			password: "PASSWORD_PRODUCTION",
		}
	}
}

// getKafkaBrokers retrieves Kafka broker addresses from environment variables based on the config.
// It handles the mapping between GO_ENV, KAFKA_MODELS, and the actual environment variable names.
func getKafkaBrokers(conf *Config) []string {
	env := conf.App.Env
	kafkaModel := initKafkaModels(conf)

	var brokerEnv string
	switch {
	case kafkaModel == "kafka-production" && env == "production":
		brokerEnv = "KAFKA_BROKERS_PRODUCTION"
	case kafkaModel == "kafka-staging" && (env == "staging" || env == "localhost"):
		brokerEnv = "KAFKA_BROKERS_STAGING"
	default:
		return []string{}
	}

	if broker, exists := os.LookupEnv(brokerEnv); exists && broker != "" {
		return strings.Split(broker, ",")
	}

	return []string{}
}

// initKafkaModels initializes the Kafka model configuration from the KAFKA_MODELS environment variable.
func initKafkaModels(conf *Config) string {
	conf.App.KafkaModels = getEnvValue(kafkaEnvConfig)
	return conf.App.KafkaModels
}

// initApp initializes global application settings like environment, log prefix, and log mode.
func initApp(conf *Config) {
	conf.App.Env = getEnvValue(appEnvConfig)
	conf.App.LogFilePrefix = getEnvValue(logFilePrefixEnvConfig)
	conf.App.LogOutputMode = getEnvValue(logOutputModeEnvConfig)
}

// initKafka initializes the Kafka brokers or Confluent Cloud configuration based on the selected model.
func initKafka(conf *Config) {
	if conf.App.KafkaModels == "kafka-confluent" {
		initKafkaConfluent(conf)
		return
	}
	conf.Kafka.Brokers = getKafkaBrokers(conf)
}

// initOracle initializes the Oracle database connection details from environment variables.
func initOracle(conf *Config) {
	env := getOracleEnv(conf.App.Env)
	conf.Oracle.Username = os.Getenv(env.username)
	conf.Oracle.Password = os.Getenv(env.password)
	conf.Oracle.Host = os.Getenv(env.host)
	conf.Oracle.Port = os.Getenv(env.port)
	conf.Oracle.DatabaseName = os.Getenv(env.database)
}

// initOperation initializes the custom operation mode from the OPERATION environment variable.
func initOperation(conf *Config) {
	conf.App.Operation = getEnvValue(operationEnvConfig)
}

// initKafkaConfluent initializes the Confluent Cloud Kafka credentials and server details.
func initKafkaConfluent(conf *Config) {
	conf.KafkaConfluent.ApiKey = os.Getenv("API_KEY_KAFKA_CONFLUENT")
	conf.KafkaConfluent.ApiSecret = os.Getenv("API_SECRET_KAFKA_CONFLUENT")
	conf.KafkaConfluent.Resource = os.Getenv("RESOURCE_KAFKA_CONFLUENT")

	if bootstrapServer, exists := os.LookupEnv("BOOTSTRAP_SERVER_KAFKA_CONFLUENT"); exists && bootstrapServer != "" {
		conf.KafkaConfluent.BootstrapServers = []string{bootstrapServer}
	}
}

// initElasticSearch initializes the Elasticsearch connection details from environment variables.
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
// IMPORTANT: This function must be called during program initialization (e.g., in init()),
// BEFORE calling NewConfig(). It is NOT safe to call concurrently or after config has
// been initialized, as it modifies package-level state without synchronization.
//
// Example:
//
//	func init() {
//	    nanopony.WithOperationValidValues([]string{"read-only", "write", "admin"})
//	}
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
