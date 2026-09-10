package nanopony

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

// MessageHandler defines the handler for processing consumed messages.
// It receives the raw message bytes and returns an error if processing fails.
type MessageHandler func(message []byte) error

// KafkaConsumer implements a native Kafka consumer using pure Go sockets.
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
	config     KafkaConsumerConfig
	retryDelay time.Duration

	mu     sync.Mutex
	conn   *KafkaConn
	offset int64
	closed bool
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
	// Use FirstOffset or LastOffset. Defaults to LastOffset.
	StartOffset int64
	// RetryDelay is the delay before retrying after a handler error.
	// Default is 1 second. Set to 0 to disable (immediate retry).
	RetryDelay time.Duration
	// Transport holds transport security settings (TLS/SASL)
	Transport *KafkaTransport
}

// NewKafkaConsumer creates a new Kafka consumer with the given configuration.
// If StartOffset is 0, it defaults to LastOffset.
// If RetryDelay is 0, it defaults to 1 second backoff on handler errors.
func NewKafkaConsumer(config KafkaConsumerConfig) *KafkaConsumer {
	startOffset := config.StartOffset
	if startOffset == 0 {
		startOffset = LastOffset
	}

	retryDelay := config.RetryDelay
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	return &KafkaConsumer{
		config:     config,
		retryDelay: retryDelay,
		offset:     startOffset,
	}
}

// ConsumeWithContext starts consuming messages with context support.
// This is a blocking call that runs until the context is cancelled.
//
// Message processing flow:
// 1. Fetch message from Kafka
// 2. Call handler with message value
// 3. If handler succeeds, advance offset
// 4. If handler fails, wait for RetryDelay before retry (default 1s backoff)
func (c *KafkaConsumer) ConsumeWithContext(ctx context.Context, handler MessageHandler) error {
	if len(c.config.Brokers) == 0 {
		return errors.New("kafka consumer has no brokers configured")
	}

	broker := c.config.Brokers[0]
	conn, err := DialKafka(ctx, broker, c.config.Transport)
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	defer func() {
		_ = c.Close()
	}()

	// Resolve initial offset if set to FirstOffset or LastOffset
	if c.offset < 0 {
		resolvedOffset, err := conn.GetOffset(c.config.Topic, 0, c.offset)
		if err == nil && resolvedOffset >= 0 {
			c.offset = resolvedOffset
		} else {
			c.offset = 0
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			messages, _, err := conn.FetchMessages(c.config.Topic, 0, c.offset, 1048576)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// Retry fetching after a brief backoff
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(500 * time.Millisecond):
					continue
				}
			}

			if len(messages) == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(200 * time.Millisecond):
					continue
				}
			}

			for _, msg := range messages {
				if err := handler(msg.Value); err != nil {
					LogFramework("WARNING", "KafkaConsumer", fmt.Sprintf("handler error: %v (retrying after %v)", err, c.retryDelay))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(c.retryDelay):
					}
					// Do not advance offset on error, retry the same message
					break
				}
				c.offset = msg.Offset + 1
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

// Close closes the consumer and releases network resources
func (c *KafkaConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
