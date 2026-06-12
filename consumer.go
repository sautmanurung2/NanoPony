package nanopony

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// MessageHandler defines the handler for processing consumed messages.
type MessageHandler func(message []byte) error

// KafkaConsumer implements a Kafka consumer using kafka-go Reader.
// Beginner Note: A Consumer "listens" to a Kafka topic and performs an
// action whenever a new message arrives.
type KafkaConsumer struct {
	reader     *kafka.Reader
	retryDelay time.Duration
	logger     *LogManager
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
	StartOffset int64
	// RetryDelay is the delay before retrying after a handler error.
	RetryDelay time.Duration
}

// NewKafkaConsumer creates a new Kafka consumer with the given configuration.
func NewKafkaConsumer(config KafkaConsumerConfig) *KafkaConsumer {
	readerConfig := kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     config.GroupID,
		StartOffset: config.StartOffset,
	}

	if config.StartOffset == 0 {
		readerConfig.StartOffset = kafka.LastOffset
	}

	return &KafkaConsumer{
		reader:     kafka.NewReader(readerConfig),
		retryDelay: config.RetryDelay,
	}
}

// WithLogger sets the logger for the consumer.
func (c *KafkaConsumer) WithLogger(lm *LogManager) *KafkaConsumer {
	c.logger = lm
	return c
}

// ConsumeWithContext starts consuming messages with context support.
func (c *KafkaConsumer) ConsumeWithContext(ctx context.Context, handler MessageHandler) error {
	retryDelay := c.retryDelay
	if retryDelay <= 0 {
		retryDelay = time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("error reading message: %w", err)
			}

			if err := handler(msg.Value); err != nil {
				if c.logger != nil {
					c.logger.LogFramework("WARNING", "KafkaConsumer", fmt.Sprintf("handler error: %v (retrying after %v)", err, retryDelay))
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay):
					continue
				}
			}
		}
	}
}

// MessageHandlerProto defines the handler for processing consumed protobuf messages.
type MessageHandlerProto func(msg proto.Message) error

// ConsumeWithContextProto consumes messages and unmarshals them from Protobuf.
func (c *KafkaConsumer) ConsumeWithContextProto(
	ctx context.Context,
	newMsg func() proto.Message,
	handler MessageHandlerProto,
) error {
	return c.ConsumeWithContext(ctx, func(message []byte) error {
		msg := newMsg()
		if err := proto.Unmarshal(message, msg); err != nil {
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
