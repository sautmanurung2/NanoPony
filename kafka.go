package nanopony

import (
	"crypto/tls"
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
	return &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Balancer:     config.Balancer,
		BatchTimeout: config.BatchTimeout,
		Transport:    config.Transport,
	}
}

// NewKafkaWriterFromConfig creates Kafka writer from Config
func NewKafkaWriterFromConfig(conf *Config) *kafka.Writer {
	config := DefaultKafkaWriterConfig()

	if conf.App.KafkaModels == "kafka-confluent" {
		kconf := conf.EnsureKafkaConfluent()
		config.Brokers = kconf.BootstrapServers
		writer := &kafka.Writer{
			Addr:         kafka.TCP(config.Brokers...),
			Balancer:     config.Balancer,
			BatchTimeout: config.BatchTimeout,
			Transport:    createSASLTransport(kconf.ApiKey, kconf.ApiSecret),
		}
		return writer
	} else {
		kconf := conf.EnsureKafka()
		config.Brokers = kconf.Brokers
		writer := &kafka.Writer{
			Addr:         kafka.TCP(config.Brokers...),
			Balancer:     config.Balancer,
			BatchTimeout: config.BatchTimeout,
		}
		return writer
	}
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
