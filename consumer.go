package nanopony

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// MessageHandler defines the handler for processing consumed messages.
// It receives the raw message bytes and returns an error if processing fails.
type MessageHandler func(message []byte) error

// ProtoMessageHandler is a generic handler for Protobuf messages.
type ProtoMessageHandler[T proto.Message] func(message T) error

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
	reader     *kafka.Reader
	retryDelay time.Duration
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
	// RetryDelay is the delay before retrying after a handler error.
	// Default is 1 second. Set to 0 to disable (immediate retry).
	RetryDelay time.Duration
}

// NewKafkaConsumer creates a new Kafka consumer with the given configuration.
// If StartOffset is 0, it defaults to LastOffset.
// If RetryDelay is 0, it defaults to 1 second backoff on handler errors.
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
		reader:     kafka.NewReader(readerConfig),
		retryDelay: config.RetryDelay,
	}
}

// ConsumeWithContext starts consuming messages with context support.
// This is a blocking call that runs until the context is cancelled.
//
// Message processing flow:
// 1. Read message from Kafka
// 2. Call handler with message value
// 3. If handler succeeds, commit the message
// 4. If handler fails, wait for RetryDelay before retry (default 1s backoff)
//
// Note: If the handler returns an error, the message is NOT committed
// and will be re-delivered on the next consumption cycle.
func (c *KafkaConsumer) ConsumeWithContext(ctx context.Context, handler MessageHandler) error {
	// Default retry delay if not configured
	retryDelay := c.retryDelay
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

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
				// Log error and apply backoff before retry to prevent tight loops
				LogFramework("WARNING", "KafkaConsumer", fmt.Sprintf("handler error: %v (retrying after %v)", err, retryDelay))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay):
					// Backoff complete, continue to retry
				}
				continue
			}

			// Commit the message after successful processing
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("failed to commit message: %w", err)
			}
		}
	}
}

// ConsumeWithContextProto starts consuming messages, unmarshaling them into the provided proto.Message type.
// The factory function should return a new instance of the target proto message.
func (c *KafkaConsumer) ConsumeWithContextProto(ctx context.Context, factory func() proto.Message, handler func(proto.Message) error) error {
	return c.ConsumeWithContext(ctx, func(data []byte) error {
		msg := factory()
		if err := proto.Unmarshal(data, msg); err != nil {
			return fmt.Errorf("failed to unmarshal proto message: %w", err)
		}
		return handler(msg)
	})
}

// Close closes the consumer and releases resources
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
