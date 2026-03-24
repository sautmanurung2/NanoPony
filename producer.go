package nanopony

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// MessageProducer defines the interface for producing messages to Kafka
type MessageProducer interface {
	Produce(topic string, message interface{}) (bool, error)
	ProduceWithContext(ctx context.Context, topic string, message interface{}) (bool, error)
	Close() error
}

// KafkaProducer implements MessageProducer using kafka-go Writer
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(writer *kafka.Writer) *KafkaProducer {
	return &KafkaProducer{
		writer: writer,
	}
}

// Produce sends a message to the specified topic
func (p *KafkaProducer) Produce(topic string, message interface{}) (bool, error) {
	return p.ProduceWithContext(context.Background(), topic, message)
}

// ProduceWithContext sends a message with context support
func (p *KafkaProducer) ProduceWithContext(ctx context.Context, topic string, message interface{}) (bool, error) {
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

// Close closes the producer
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// MessageHandler defines the handler for processing consumed messages
type MessageHandler func(message []byte) error

// KafkaConsumer implements MessageConsumer using kafka-go Reader
type KafkaConsumer struct {
	reader *kafka.Reader
}

// KafkaConsumerConfig holds configuration for creating a consumer
type KafkaConsumerConfig struct {
	Brokers     []string
	Topic       string
	GroupID     string
	StartOffset int64
}

// NewKafkaConsumer creates a new Kafka consumer
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
		reader: kafka.NewReader(readerConfig),
	}
}

// ConsumeWithContext starts consuming messages with context support
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

			if err := handler(msg.Value); err != nil {
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("failed to commit message: %w", err)
			}
		}
	}
}

// Close closes the consumer
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
