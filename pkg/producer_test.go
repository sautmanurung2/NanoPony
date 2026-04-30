package nanopony

import (
	"testing"
)

func TestKafkaProducer(t *testing.T) {
	// Note: This test requires a running Kafka instance
	// For unit testing, we'll test the interface and structure
	t.Skip("Skipping test - requires Kafka instance")
}

func TestKafkaConsumer(t *testing.T) {
	// Note: This test requires a running Kafka instance
	t.Skip("Skipping test - requires Kafka instance")
}

func TestMessageHandler(t *testing.T) {
	handler := MessageHandler(func(message []byte) error {
		if len(message) == 0 {
			return nil
		}
		return nil
	})

	err := handler([]byte("test message"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestKafkaConsumerConfig(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "test-topic",
		GroupID:     "test-group",
		StartOffset: 0,
	}

	consumer := NewKafkaConsumer(config)
	if consumer == nil {
		t.Fatal("Expected consumer to be created")
	}
	defer consumer.Close()
}
