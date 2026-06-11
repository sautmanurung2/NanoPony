package nanopony

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestKafkaConsumerMethods(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers:     []string{"localhost:1"}, // Invalid addr
		Topic:       "test-topic",
		GroupID:     "test-group",
		RetryDelay:  1 * time.Millisecond,
	}

	consumer := NewKafkaConsumer(config)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test ConsumeWithContext - should exit on context timeout or read error
	err := consumer.ConsumeWithContext(ctx, func(message []byte) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error from ConsumeWithContext with invalid Kafka address")
	}

	// Test Close
	if err := consumer.Close(); err != nil {
		t.Errorf("Unexpected error closing consumer: %v", err)
	}
}

func TestKafkaConsumerProto(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers: []string{"localhost:1"},
		Topic:   "test",
	}
	consumer := NewKafkaConsumer(config)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := consumer.ConsumeWithContextProto(ctx, 
		func() proto.Message { return &emptypb.Empty{} },
		func(msg proto.Message) error { return nil },
	)
	if err == nil {
		t.Error("Expected error")
	}
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

func TestConsumerWithLogger(t *testing.T) {
	config := NewConfig()
	logger := NewLogManager(config)
	c := &KafkaConsumer{}
	c.WithLogger(logger)
	if c.logger == nil { t.Errorf("WithLogger failed") }
}

func TestConsumerClose(t *testing.T) {
	c := &KafkaConsumer{}
	// Calling close on uninitialized reader
	c.Close()
}
