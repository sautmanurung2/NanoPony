package nanopony

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// KafkaWriterConfig holds Kafka writer configuration
type KafkaWriterConfig struct {
	Brokers      []string
	Balancer     kafka.Balancer
	BatchTimeout time.Duration
	Transport    *kafka.Transport
}

// KafkaMessageMetadata holds metadata for a Kafka message, primarily for logging.
type KafkaMessageMetadata struct {
	LoggerEntry *LoggerEntry
	Payload     any
	LogData     string
}

// DefaultKafkaWriterConfig returns default Kafka writer configuration
func DefaultKafkaWriterConfig() KafkaWriterConfig {
	return KafkaWriterConfig{
		Balancer:     &kafka.RoundRobin{},
		BatchTimeout: 10 * time.Millisecond,
		Transport:    nil,
	}
}

// NewKafkaWriter creates a new Kafka writer with the given configuration
func NewKafkaWriter(config KafkaWriterConfig) *kafka.Writer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Balancer:     config.Balancer,
		BatchTimeout: config.BatchTimeout,
		Completion: func(messages []kafka.Message, err error) {
			for _, msg := range messages {
				meta, ok := msg.WriterData.(KafkaMessageMetadata)
				if !ok || meta.LoggerEntry == nil {
					if err != nil {
						fmt.Printf("[Kafka-Async-Error] Gagal mengirim pesan ke topic %s. Error: %v\n", msg.Topic, err)
					}
					continue
				}

				if err != nil {
					meta.LoggerEntry.LoggingData("error", meta.Payload, ResponseLog{
						Status:  "error",
						Message: fmt.Sprintf("Kafka produce error to topic %s: %v", msg.Topic, err),
					})
				} else {
					meta.LoggerEntry.LoggingData("info", meta.Payload, ResponseLog{
						Status:  "success",
						Message: fmt.Sprintf("message sent to topic : %s and data : %s", msg.Topic, meta.LogData),
					})
				}
			}
		},
	}

	if config.Transport != nil {
		w.Transport = config.Transport
	}

	return w
}

// NewKafkaWriterFromConfig creates Kafka writer from Config.
// Reuses NewKafkaWriter to avoid duplicating the writer construction.
func NewKafkaWriterFromConfig(conf *Config) *kafka.Writer {
	config := DefaultKafkaWriterConfig()

	if conf.App.KafkaModels == "kafka-confluent" {
		kconf := conf.EnsureKafkaConfluent()
		config.Brokers = kconf.BootstrapServers
		config.Transport = createSASLTransport(kconf.ApiKey, kconf.ApiSecret)
	} else {
		config.Brokers = conf.EnsureKafka().Brokers
	}

	return NewKafkaWriter(config)
}

// createSASLTransport creates a SASL/TLS transport for Confluent Cloud
func createSASLTransport(apiKey, apiSecret string) *kafka.Transport {
	return &kafka.Transport{
		TLS: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		SASL: plain.Mechanism{
			Username: apiKey,
			Password: apiSecret,
		},
	}
}

// CloseKafkaWriter safely closes a Kafka writer
func CloseKafkaWriter(writer *kafka.Writer) error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}
