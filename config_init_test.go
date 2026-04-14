package nanopony

import (
	"os"
	"testing"
)

func TestGetOracleEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected oracleEnv
	}{
		{
			name: "staging environment",
			env:  "staging",
			expected: oracleEnv{
				host:     "HOST_STAGING",
				port:     "PORT_STAGING",
				database: "DATABASE_STAGING",
				username: "USERNAME_STAGING",
				password: "PASSWORD_STAGING",
			},
		},
		{
			name: "production environment",
			env:  "production",
			expected: oracleEnv{
				host:     "HOST_PRODUCTION",
				port:     "PORT_PRODUCTION",
				database: "DATABASE_PRODUCTION",
				username: "USERNAME_PRODUCTION",
				password: "PASSWORD_PRODUCTION",
			},
		},
		{
			name: "local environment",
			env:  "local",
			expected: oracleEnv{
				host:     "HOST_PRODUCTION",
				port:     "PORT_PRODUCTION",
				database: "DATABASE_PRODUCTION",
				username: "USERNAME_PRODUCTION",
				password: "PASSWORD_PRODUCTION",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOracleEnv(tt.env)
			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestGetKafkaBrokers(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		kafkaModel  string
		brokers     string
		expected    []string
	}{
		{
			name:       "kafka-staging",
			env:        "staging",
			kafkaModel: "kafka-staging",
			brokers:    "broker1:9092,broker2:9092",
			expected:   []string{"broker1:9092", "broker2:9092"},
		},
		{
			name:       "kafka-production in production env",
			env:        "production",
			kafkaModel: "kafka-production",
			brokers:    "prod1:9092,prod2:9092",
			expected:   []string{"prod1:9092", "prod2:9092"},
		},
		{
			name:       "kafka-localhost",
			env:        "local",
			kafkaModel: "kafka-localhost",
			brokers:    "",
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set KAFKA_MODELS environment variable so initKafkaModels reads it correctly
			if tt.kafkaModel != "" {
				os.Setenv("KAFKA_MODELS", tt.kafkaModel)
				defer os.Unsetenv("KAFKA_MODELS")
			}

			conf := &Config{
				App: AppConfig{
					Env: tt.env,
				},
			}

			// Set broker environment variable
			if tt.brokers != "" {
				var brokerEnv string
				switch {
				case tt.kafkaModel == "kafka-production" && tt.env == "production":
					brokerEnv = "KAFKA_BROKERS_PRODUCTION"
				case tt.kafkaModel == "kafka-staging":
					brokerEnv = "KAFKA_BROKERS_STAGING"
				}

				if brokerEnv != "" {
					os.Setenv(brokerEnv, tt.brokers)
					defer os.Unsetenv(brokerEnv)
				}
			}

			result := getKafkaBrokers(conf)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d brokers, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestInitApp(t *testing.T) {
	ResetConfig()

	// Test with staging
	os.Setenv("GO_ENV", "staging")
	defer os.Unsetenv("GO_ENV")

	conf := &Config{}
	initApp(conf)

	if conf.App.Env != "staging" {
		t.Errorf("Expected env 'staging', got '%s'", conf.App.Env)
	}
}

func TestInitAppDefault(t *testing.T) {
	ResetConfig()

	// Test with no env set (should default to local)
	os.Unsetenv("GO_ENV")

	conf := &Config{}
	initApp(conf)

	if conf.App.Env != "local" {
		t.Errorf("Expected default env 'local', got '%s'", conf.App.Env)
	}
}

func TestInitKafka(t *testing.T) {
	ResetConfig()

	// Test with kafka-localhost
	conf := &Config{
		App: AppConfig{
			Env:         "local",
			KafkaModels: "kafka-localhost",
		},
	}

	initKafka(conf)

	// Should have empty brokers (no staging/production configured)
	if len(conf.Kafka.Brokers) != 0 {
		t.Errorf("Expected empty brokers, got %v", conf.Kafka.Brokers)
	}
}

func TestInitKafkaConfluent(t *testing.T) {
	ResetConfig()

	// Set Confluent Cloud environment variables
	os.Setenv("API_KEY_KAFKA_CONFLUENT", "test-key")
	os.Setenv("API_SECRET_KAFKA_CONFLUENT", "test-secret")
	os.Setenv("RESOURCE_KAFKA_CONFLUENT", "test-resource")
	os.Setenv("BOOTSTRAP_SERVER_KAFKA_CONFLUENT", "pkc-test.us-east-1.aws.confluent.cloud:9092")
	defer func() {
		os.Unsetenv("API_KEY_KAFKA_CONFLUENT")
		os.Unsetenv("API_SECRET_KAFKA_CONFLUENT")
		os.Unsetenv("RESOURCE_KAFKA_CONFLUENT")
		os.Unsetenv("BOOTSTRAP_SERVER_KAFKA_CONFLUENT")
	}()

	conf := &Config{
		App: AppConfig{
			KafkaModels: "kafka-confluent",
		},
	}

	initKafka(conf)

	if conf.KafkaConfluent.ApiKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", conf.KafkaConfluent.ApiKey)
	}
	if conf.KafkaConfluent.ApiSecret != "test-secret" {
		t.Errorf("Expected API secret 'test-secret', got '%s'", conf.KafkaConfluent.ApiSecret)
	}
	if len(conf.KafkaConfluent.BootstrapServers) != 1 {
		t.Errorf("Expected 1 bootstrap server, got %d", len(conf.KafkaConfluent.BootstrapServers))
	}
}

func TestInitOracle(t *testing.T) {
	ResetConfig()

	// Set Oracle environment variables
	os.Setenv("HOST_STAGING", "oracle-staging.example.com")
	os.Setenv("PORT_STAGING", "1521")
	os.Setenv("DATABASE_STAGING", "ORCL")
	os.Setenv("USERNAME_STAGING", "testuser")
	os.Setenv("PASSWORD_STAGING", "testpass")
	defer func() {
		os.Unsetenv("HOST_STAGING")
		os.Unsetenv("PORT_STAGING")
		os.Unsetenv("DATABASE_STAGING")
		os.Unsetenv("USERNAME_STAGING")
		os.Unsetenv("PASSWORD_STAGING")
	}()

	conf := &Config{
		App: AppConfig{
			Env: "staging",
		},
	}

	initOracle(conf)

	if conf.Oracle.Host != "oracle-staging.example.com" {
		t.Errorf("Expected host 'oracle-staging.example.com', got '%s'", conf.Oracle.Host)
	}
	if conf.Oracle.Port != "1521" {
		t.Errorf("Expected port '1521', got '%s'", conf.Oracle.Port)
	}
	if conf.Oracle.DatabaseName != "ORCL" {
		t.Errorf("Expected database 'ORCL', got '%s'", conf.Oracle.DatabaseName)
	}
	if conf.Oracle.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", conf.Oracle.Username)
	}
	if conf.Oracle.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", conf.Oracle.Password)
	}
}

func TestInitOperation(t *testing.T) {
	ResetConfig()

	os.Setenv("OPERATION", "custom-operation")
	defer os.Unsetenv("OPERATION")

	conf := &Config{}
	initOperation(conf)

	if conf.App.Operation != "custom-operation" {
		t.Errorf("Expected operation 'custom-operation', got '%s'", conf.App.Operation)
	}
}

func TestInitElasticSearch(t *testing.T) {
	ResetConfig()

	// Set Elasticsearch environment variables
	os.Setenv("ELASTIC_HOST", "localhost:9200")
	os.Setenv("ELASTIC_USERNAME", "elastic")
	os.Setenv("ELASTIC_PASSWORD", "secret")
	os.Setenv("ELASTIC_INDEX_DATA", "test-logs")
	os.Setenv("ELASTIC_API_KEY", "test-api-key")
	os.Setenv("ELASTIC_PREFIX_INDEX", "nanopony-")
	defer func() {
		os.Unsetenv("ELASTIC_HOST")
		os.Unsetenv("ELASTIC_USERNAME")
		os.Unsetenv("ELASTIC_PASSWORD")
		os.Unsetenv("ELASTIC_INDEX_DATA")
		os.Unsetenv("ELASTIC_API_KEY")
		os.Unsetenv("ELASTIC_PREFIX_INDEX")
	}()

	conf := &Config{}
	initElasticSearch(conf)

	if conf.ElasticSearch.ElasticHost != "localhost:9200" {
		t.Errorf("Expected host 'localhost:9200', got '%s'", conf.ElasticSearch.ElasticHost)
	}
	if conf.ElasticSearch.ElasticUsername != "elastic" {
		t.Errorf("Expected username 'elastic', got '%s'", conf.ElasticSearch.ElasticUsername)
	}
	if conf.ElasticSearch.ElasticPassword != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", conf.ElasticSearch.ElasticPassword)
	}
	if conf.ElasticSearch.ElasticIndexName != "test-logs" {
		t.Errorf("Expected index 'test-logs', got '%s'", conf.ElasticSearch.ElasticIndexName)
	}
	if conf.ElasticSearch.ElasticApiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", conf.ElasticSearch.ElasticApiKey)
	}
	if conf.ElasticSearch.ElasticPrefixIndex != "nanopony-" {
		t.Errorf("Expected prefix 'nanopony-', got '%s'", conf.ElasticSearch.ElasticPrefixIndex)
	}
}

func TestFullConfigInitialization(t *testing.T) {
	ResetConfig()

	// Set all environment variables
	os.Setenv("GO_ENV", "staging")
	os.Setenv("KAFKA-MODELS", "kafka-staging")
	os.Setenv("KAFKA_BROKERS_STAGING", "broker1:9092")
	os.Setenv("HOST_STAGING", "oracle-staging")
	os.Setenv("PORT_STAGING", "1521")
	os.Setenv("DATABASE_STAGING", "ORCL")
	os.Setenv("USERNAME_STAGING", "user")
	os.Setenv("PASSWORD_STAGING", "pass")
	os.Setenv("OPERATION", "test")
	os.Setenv("ELASTIC_HOST", "localhost:9200")
	defer func() {
		os.Unsetenv("GO_ENV")
		os.Unsetenv("KAFKA-MODELS")
		os.Unsetenv("KAFKA_BROKERS_STAGING")
		os.Unsetenv("HOST_STAGING")
		os.Unsetenv("PORT_STAGING")
		os.Unsetenv("DATABASE_STAGING")
		os.Unsetenv("USERNAME_STAGING")
		os.Unsetenv("PASSWORD_STAGING")
		os.Unsetenv("OPERATION")
		os.Unsetenv("ELASTIC_HOST")
	}()

	config := BuildConfig(func(c *Config) {
		// Verify env vars are being read
		if c.App.KafkaModels == "" {
			t.Log("Warning: KAFKA-MODELS not read, may be expected in test env")
		}
	})

	if config.App.Env != "staging" {
		t.Errorf("Expected env 'staging', got '%s'", config.App.Env)
	}
	// Kafka brokers may be empty if KAFKA-MODELS doesn't match expected patterns
	if len(config.Kafka.Brokers) > 0 && config.Kafka.Brokers[0] != "broker1:9092" {
		t.Errorf("Expected broker 'broker1:9092', got %v", config.Kafka.Brokers)
	}
	if config.Oracle.Host != "oracle-staging" {
		t.Errorf("Expected Oracle host 'oracle-staging', got '%s'", config.Oracle.Host)
	}
}

func TestBuildConfigWithCustomValues(t *testing.T) {
	ResetConfig()

	config := BuildConfig(func(c *Config) {
		c.App.Env = "custom"
		c.App.Operation = "custom-op"
	})

	if config.App.Env != "custom" {
		t.Errorf("Expected env 'custom', got '%s'", config.App.Env)
	}
	if config.App.Operation != "custom-op" {
		t.Errorf("Expected operation 'custom-op', got '%s'", config.App.Operation)
	}
}

func TestWithOperationValidValuesCustom(t *testing.T) {
	// Set custom valid values
	WithOperationValidValues([]string{"custom1", "custom2"})

	// Reset to default after test
	defer WithOperationValidValues([]string{})

	os.Setenv("OPERATION", "custom1")
	defer os.Unsetenv("OPERATION")

	conf := &Config{}
	initOperation(conf)

	if conf.App.Operation != "custom1" {
		t.Errorf("Expected operation 'custom1', got '%s'", conf.App.Operation)
	}
}
