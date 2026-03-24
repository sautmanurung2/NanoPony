package nanopony

import (
	"os"
	"slices"
	"strings"
)

type envConfig struct {
	name        string
	validValues []string
	defaultVal  string
}

type oracleEnv struct {
	host     string
	port     string
	database string
	username string
	password string
}

var (
	appEnvConfig = envConfig{
		name:        "GO_ENV",
		validValues: []string{"staging", "production"},
		defaultVal:  "local",
	}

	kafkaEnvConfig = envConfig{
		name:        "KAFKA-MODELS",
		validValues: []string{"kafka-staging", "kafka-production", "kafka-confluent"},
		defaultVal:  "kafka-localhost",
	}

	operationEnvConfig = envConfig{
		name:        "OPERATION",
		validValues: []string{},
		defaultVal:  "default",
	}
)

// getEnvValue retrieves an environment variable value with validation
func getEnvValue(cfg envConfig) string {
	value := strings.ToLower(os.Getenv(cfg.name))

	if len(cfg.validValues) == 0 {
		return value
	}

	if slices.Contains(cfg.validValues, value) {
		return value
	}

	return cfg.defaultVal
}

// getOracleEnv returns Oracle environment variable names based on environment
func getOracleEnv(env string) oracleEnv {
	if env == "staging" {
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

// getKafkaBrokers retrieves Kafka broker addresses from environment
func getKafkaBrokers(conf *Config) []string {
	env := conf.App.Env
	kafkaModel := conf.App.KafkaModels

	var brokerEnv string
	switch {
	case kafkaModel == "kafka-production" && env == "production":
		brokerEnv = "KAFKA_BROKERS_PRODUCTION"
	case kafkaModel == "kafka-staging":
		brokerEnv = "KAFKA_BROKERS_STAGING"
	default:
		return []string{}
	}

	if broker, exists := os.LookupEnv(brokerEnv); exists && broker != "" {
		return strings.Split(broker, ",")
	}

	return []string{}
}

// initApp initializes application configuration
func initApp(conf *Config) {
	conf.App.Env = getEnvValue(appEnvConfig)
}

// initKafka initializes Kafka configuration
func initKafka(conf *Config) {
	if conf.App.KafkaModels == "kafka-confluent" {
		initKafkaConfluent(conf)
		return
	}
	conf.Kafka.Brokers = getKafkaBrokers(conf)
}

// initOracle initializes Oracle database configuration
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

// initKafkaConfluent initializes Confluent Cloud Kafka configuration
func initKafkaConfluent(conf *Config) {
	conf.KafkaConfluent.ApiKey = os.Getenv("API_KEY_KAFKA_CONFLUENT")
	conf.KafkaConfluent.ApiSecret = os.Getenv("API_SECRET_KAFKA_CONFLUENT")
	conf.KafkaConfluent.Resource = os.Getenv("RESOURCE_KAFKA_CONFLUENT")

	if bootstrapServer, exists := os.LookupEnv("BOOTSTRAP_SERVER_KAFKA_CONFLUENT"); exists && bootstrapServer != "" {
		conf.KafkaConfluent.BootstrapServers = []string{bootstrapServer}
	}
}

// WithOperationValidValues sets valid values for operation environment variable.
// This is useful for custom operation modes.
func WithOperationValidValues(values []string) {
	operationEnvConfig.validValues = values
}
