package nanopony

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestKafkaProducerProtobuf(t *testing.T) {
	writer := &kafka.Writer{
		Addr: kafka.TCP("localhost:1"),
	}
	producer := NewKafkaProducer(writer)
	msg := &emptypb.Empty{}

	// Test ProduceProto
	_, err := producer.ProduceProto("test", msg, nil)
	if err == nil {
		t.Error("Expected error for invalid Kafka address")
	}

	// Test ProduceProtoWithKey
	_, err = producer.ProduceProtoWithKey("test", []byte("key"), msg, nil)
	if err == nil {
		t.Error("Expected error for invalid Kafka address")
	}

	// Test ProduceWithContextProto
	_, err = producer.ProduceWithContextProto(context.Background(), "test", msg, nil)
	if err == nil {
		t.Error("Expected error for invalid Kafka address")
	}

	// Test ProduceWithContextAndKeyProto
	_, err = producer.ProduceWithContextAndKeyProto(context.Background(), "test", []byte("key"), msg, nil)
	if err == nil {
		t.Error("Expected error for invalid Kafka address")
	}
}

func TestKafkaProducerMarshaling(t *testing.T) {
	// Test with a writer that will fail if it tries to send
	writer := &kafka.Writer{
		Addr:   kafka.TCP("localhost:1"), // Invalid addr
		Topic:  "test-topic",
		Async:  false,
	}
	producer := NewKafkaProducer(writer)

	logger := NewLoggerFromOptions(LoggerOptions{ServiceName: "test"})

	// Test Produce (JSON)
	_, err := producer.Produce("test-topic", map[string]string{"key": "value"}, logger)
	if err == nil {
		t.Error("Expected error for invalid Kafka address")
	}
}

func TestKafkaProducerClose(t *testing.T) {
	writer := &kafka.Writer{}
	producer := NewKafkaProducer(writer)
	if err := producer.Close(); err != nil {
		t.Errorf("Expected nil error from Close, got %v", err)
	}
}

func TestKafkaProducerNilLogger(t *testing.T) {
	writer := &kafka.Writer{Addr: kafka.TCP("localhost:1")}
	producer := NewKafkaProducer(writer)
	_, _ = producer.Produce("test", "msg", nil) // Should not panic
}
