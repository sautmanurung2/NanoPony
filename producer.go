package nanopony

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// MessageProducer defines the interface for producing messages to Kafka.
// This abstraction allows for easy testing and swapping implementations.
//
// Example:
//
//	producer := NewKafkaProducer(writer)
//	success, err := producer.Produce("my-topic", map[string]interface{}{
//	    "id":   1,
//	    "data": "hello",
//	})
type MessageProducer interface {
	// Produce sends a message to the specified topic with background context
	Produce(topic string, message any) (bool, error)
	// ProduceWithContext sends a message with context support for cancellation and timeout
	ProduceWithContext(ctx context.Context, topic string, message any) (bool, error)
	// Close closes the producer and releases resources
	Close() error
}

// KafkaProducer implements MessageProducer using kafka-go Writer.
// It provides JSON serialization for messages.
//
// Example:
//
//	writer := NewKafkaWriterFromConfig(config)
//	producer := NewKafkaProducer(writer)
//	defer producer.Close()
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a new Kafka producer from an existing writer.
func NewKafkaProducer(writer *kafka.Writer) *KafkaProducer {
	return &KafkaProducer{
		writer: writer,
	}
}

// Produce sends a message to the specified topic using background context.
// Returns true if successful, or an error if the message could not be sent.
func (p *KafkaProducer) Produce(topic string, message any) (bool, error) {
	return p.ProduceWithContext(context.Background(), topic, message)
}

// ProduceWithContext sends a message with context support for cancellation and timeout.
// The message is marshaled to JSON before sending.
func (p *KafkaProducer) ProduceWithContext(ctx context.Context, topic string, message any) (bool, error) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return false, fmt.Errorf("failed to marshal message: %w", err)
	}

	kafkaMsg := kafka.Message{
		Topic: topic,
		Value: messageBytes,
	}

	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		return false, fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return true, nil
}

// Close closes the producer and releases resources
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// MessageHandler defines the handler for processing consumed messages.
// It receives the raw message bytes and returns an error if processing fails.
type MessageHandler func(message []byte) error

// KafkaConsumer implements a Kafka consumer using kafka-go Reader.
// It provides a simple way to consume messages from a single topic.
//
// Example:
//
//	consumer := NewKafkaConsumer(KafkaConsumerConfig{
//	    Brokers: []string{"localhost:9092"},
//	    Topic:   "my-topic",
//	    GroupID: "my-group",
//	})
//	defer consumer.Close()
//
//	err := consumer.ConsumeWithContext(ctx, func(message []byte) error {
//	    log.Printf("Received: %s", message)
//	    return nil
//	})
type KafkaConsumer struct {
	reader *kafka.Reader
}

// KafkaConsumerConfig holds configuration for creating a consumer
type KafkaConsumerConfig struct {
	// Brokers is the list of Kafka broker addresses
	Brokers []string
	// Topic is the topic to consume from
	Topic string
	// GroupID is the consumer group ID
	GroupID string
	// StartOffset is the initial offset to start from.
	// Use kafka.FirstOffset or kafka.LastOffset. Defaults to LastOffset.
	StartOffset int64
}

// NewKafkaConsumer creates a new Kafka consumer with the given configuration.
// If StartOffset is 0, it defaults to LastOffset.
func NewKafkaConsumer(config KafkaConsumerConfig) *KafkaConsumer {
	readerConfig := kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     config.GroupID,
		StartOffset: config.StartOffset,
	}

	// Default to LastOffset if not specified
	if config.StartOffset == 0 {
		readerConfig.StartOffset = kafka.LastOffset
	}

	return &KafkaConsumer{
		reader: kafka.NewReader(readerConfig),
	}
}

// ConsumeWithContext starts consuming messages with context support.
// This is a blocking call that runs until the context is cancelled.
//
// Message processing flow:
// 1. Read message from Kafka
// 2. Call handler with message value
// 3. If handler succeeds, commit the message
// 4. If handler fails, skip the message (do not commit)
//
// Note: If the handler returns an error, the message is NOT committed
// and will be re-delivered on the next consumption cycle.
func (c *KafkaConsumer) ConsumeWithContext(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to read message: %w", err)
			}

			// Process the message with handler
			if err := handler(msg.Value); err != nil {
				// Handler failed - message is NOT committed
				// It will be re-delivered on next read
				continue
			}

			// Commit the message after successful processing
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("failed to commit message: %w", err)
			}
		}
	}
}

// Close closes the consumer and releases resources
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
