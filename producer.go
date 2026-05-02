package nanopony

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MessageProducer defines the interface for producing messages to Kafka.
// This abstraction allows for easy testing and swapping implementations.
//
// Each Produce call requires a *LoggerEntry for structured logging of the
// send result (success or failure). Create one via NewLogger() or NewLoggerFromOptions().
//
// Example:
//
//	producer := NewKafkaProducer(writer)
//	logger := NewLoggerFromOptions(LoggerOptions{ServiceName: "my-svc"})
//	success, err := producer.Produce("my-topic", myPayload, logger)
type MessageProducer interface {
	// Produce sends a message to the specified topic with background context.
	// The loggerEntry is used to log the outcome of the send operation.
	Produce(topic string, message any, loggerEntry *LoggerEntry) (bool, error)
	// ProduceWithContext sends a message with context support for cancellation and timeout.
	// The loggerEntry is used to log the outcome of the send operation.
	ProduceWithContext(ctx context.Context, topic string, message any, loggerEntry *LoggerEntry) (bool, error)
	// ProduceProto sends a protobuf message to the specified topic using background context.
	ProduceProto(topic string, message proto.Message, loggerEntry *LoggerEntry) (bool, error)
	// ProduceWithContextProto sends a protobuf message with context support.
	ProduceWithContextProto(ctx context.Context, topic string, message proto.Message, loggerEntry *LoggerEntry) (bool, error)
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
func (p *KafkaProducer) Produce(topic string, message any, loggerEntry *LoggerEntry) (bool, error) {
	return p.ProduceWithContext(context.Background(), topic, message, loggerEntry)
}

// ProduceProto sends a message to the specified topic using background context.
// Returns true if successful, or an error if the message could not be sent.
func (p *KafkaProducer) ProduceProto(topic string, message proto.Message, loggerEntry *LoggerEntry) (bool, error) {
	return p.ProduceWithContextProto(context.Background(), topic, message, loggerEntry)
}

// writeKafkaMessage is an internal helper that handles the common logic for sending messages
// and logging the outcomes to reduce code duplication.
func (p *KafkaProducer) writeKafkaMessage(ctx context.Context, topic string, payload any, messageBytes []byte, logData string, loggerEntry *LoggerEntry) (bool, error) {
	kafkaMsg := kafka.Message{
		Topic: topic,
		Value: messageBytes,
	}

	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		info := fmt.Sprintf("Error writing message to Kafka : %s", err)
		loggerEntry.LoggingData("error", payload, ResponseLog{
			Status:  "error",
			Message: info,
		})
		return false, fmt.Errorf("failed to write message to kafka: %w", err)
	}

	info := fmt.Sprintf("message sent to topic : %s and data : %s", topic, logData)
	loggerEntry.LoggingData("info", payload, ResponseLog{
		Status:  "success",
		Message: info,
	})

	return true, nil
}

// ProduceWithContext sends a message with context support for cancellation and timeout.
// The message is marshaled to JSON before sending.
func (p *KafkaProducer) ProduceWithContext(ctx context.Context, topic string, message any, loggerEntry *LoggerEntry) (bool, error) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return false, fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.writeKafkaMessage(ctx, topic, message, messageBytes, string(messageBytes), loggerEntry)
}

// ProduceWithContextProto sends a message with context support for cancellation and timeout.
// The message is marshaled to Proto file before sending
func (p *KafkaProducer) ProduceWithContextProto(ctx context.Context, topic string, message proto.Message, loggerEntry *LoggerEntry) (bool, error) {
	messageBytes, err := proto.Marshal(message)
	if err != nil {
		return false, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use protojson for human-readable logging of binary data
	logData, _ := protojson.Marshal(message)
	return p.writeKafkaMessage(ctx, topic, message, messageBytes, string(logData), loggerEntry)
}

// Close closes the producer and releases resources
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}


