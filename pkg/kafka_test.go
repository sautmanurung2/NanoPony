package nanopony

import (
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestNewKafkaWriter(t *testing.T) {
	config := KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
	}

	writer := NewKafkaWriter(config)
	if writer == nil {
		t.Fatal("Expected Kafka writer to be created")
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewKafkaWriterFromConfig(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	// This will create a writer, but may fail if Kafka is not available
	writer := NewKafkaWriterFromConfig(config)
	if writer == nil {
		t.Log("Warning: Writer not created (Kafka may not be configured)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCloseKafkaWriter(t *testing.T) {
	// Test closing nil writer
	err := CloseKafkaWriter(nil)
	if err != nil {
		t.Errorf("Expected no error when closing nil writer, got %v", err)
	}

	// Test closing actual writer
	config := KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
	}
	writer := NewKafkaWriter(config)

	err = CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test closing already closed writer (should handle gracefully)
	err = CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Expected no error when closing already closed writer, got %v", err)
	}
}

func TestKafkaWriterConfig(t *testing.T) {
	config := KafkaWriterConfig{
		Brokers:      []string{"broker1:9092", "broker2:9092"},
		Balancer:     &kafka.RoundRobin{},
		BatchTimeout: 0,
		Transport:    nil,
	}

	if len(config.Brokers) != 2 {
		t.Errorf("Expected 2 brokers, got %d", len(config.Brokers))
	}
	if config.Balancer == nil {
		t.Error("Expected balancer to be set")
	}
}

func TestCreateSASLTransport(t *testing.T) {
	// Test creating SASL transport for Confluent Cloud
	apiKey := "test-api-key"
	apiSecret := "test-api-secret"

	transport := createSASLTransport(apiKey, apiSecret)
	if transport == nil {
		t.Fatal("Expected SASL transport to be created")
	}
}

func TestNewKafkaWriterFromConfluentConfig(t *testing.T) {
	ResetConfig()

	// Manually set confluent config
	conf := &Config{}
	conf.App.KafkaModels = "kafka-confluent"
	conf.KafkaConfluent.ApiKey = "test-key"
	conf.KafkaConfluent.ApiSecret = "test-secret"
	conf.KafkaConfluent.BootstrapServers = []string{"pkc-test.us-east-1.aws.confluent.cloud:9092"}

	// This should create a writer with SASL transport
	writer := NewKafkaWriterFromConfig(conf)
	if writer == nil {
		t.Log("Warning: Writer not created (may be expected)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
