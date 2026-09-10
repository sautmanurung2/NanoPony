package nanopony

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// KafkaMessage holds data for a single Kafka message
type KafkaMessage struct {
	Topic      string
	Key        []byte
	Value      []byte
	WriterData any
	Partition  int
	Offset     int64
	Time       time.Time
}

// KafkaSASLPlain holds SASL PLAIN username and password
type KafkaSASLPlain struct {
	Username string
	Password string
}

// KafkaTransport holds transport security and authentication settings
type KafkaTransport struct {
	TLS  *tls.Config
	SASL *KafkaSASLPlain
}

// Balancer distributes messages among available topic partitions
type Balancer interface {
	Balance(msg KafkaMessage, partitions ...int) int
}

// RoundRobin balances messages round-robin across partitions
type RoundRobin struct {
	counter atomic.Uint32
}

// Balance implements Balancer for RoundRobin
func (r *RoundRobin) Balance(msg KafkaMessage, partitions ...int) int {
	if len(partitions) == 0 {
		return 0
	}
	idx := r.counter.Add(1) - 1
	return partitions[int(idx)%len(partitions)]
}

// Hash balances messages using FNV-1a hash of the message key
type Hash struct{}

// Balance implements Balancer for Hash
func (h *Hash) Balance(msg KafkaMessage, partitions ...int) int {
	if len(partitions) == 0 {
		return 0
	}
	if len(msg.Key) == 0 {
		return partitions[0]
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write(msg.Key)
	return partitions[int(hasher.Sum32())%len(partitions)]
}

// KafkaWriterConfig holds Kafka writer configuration
type KafkaWriterConfig struct {
	Brokers      []string
	Balancer     Balancer
	BatchTimeout time.Duration
	BatchSize    int
	Transport    *KafkaTransport
}

// KafkaMessageMetadata holds metadata for a Kafka message, primarily for logging.
type KafkaMessageMetadata struct {
	LoggerEntry *LoggerEntry
	Payload     any
	LogData     string
}

// KafkaWriter writes messages to Kafka brokers using pure native Go sockets
type KafkaWriter struct {
	Brokers      []string
	Balancer     Balancer
	BatchTimeout time.Duration
	BatchSize    int
	Transport    *KafkaTransport
	Completion   func(messages []KafkaMessage, err error)

	mu    sync.Mutex
	conns map[string]*KafkaConn
}

// DefaultKafkaWriterConfigRoundRobin returns default Kafka writer configuration with RoundRobin balancer
func DefaultKafkaWriterConfigRoundRobin() KafkaWriterConfig {
	return KafkaWriterConfig{
		Balancer:     &RoundRobin{},
		BatchTimeout: 10 * time.Millisecond,
		Transport:    nil,
	}
}

// DefaultKafkaWriterConfigHash returns default Kafka writer configuration with Hash balancer
func DefaultKafkaWriterConfigHash() KafkaWriterConfig {
	return KafkaWriterConfig{
		Balancer:     &Hash{},
		BatchTimeout: 10 * time.Millisecond,
		Transport:    nil,
	}
}

// NewKafkaWriter creates a new Kafka writer with the given configuration
func NewKafkaWriter(config KafkaWriterConfig) *KafkaWriter {
	balancer := config.Balancer
	if balancer == nil {
		balancer = &RoundRobin{}
	}

	w := &KafkaWriter{
		Brokers:      config.Brokers,
		Balancer:     balancer,
		BatchTimeout: config.BatchTimeout,
		BatchSize:    config.BatchSize,
		Transport:    config.Transport,
		conns:        make(map[string]*KafkaConn),
		Completion: func(messages []KafkaMessage, err error) {
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

	return w
}

// WriteMessages writes one or more messages to Kafka
func (w *KafkaWriter) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	if w == nil {
		return errors.New("kafka writer is nil")
	}
	if len(w.Brokers) == 0 {
		err := errors.New("kafka writer has no brokers configured")
		if w.Completion != nil {
			w.Completion(msgs, err)
		}
		return err
	}

	conn, err := w.getConn(ctx)
	if err != nil {
		if w.Completion != nil {
			w.Completion(msgs, err)
		}
		return err
	}

	// Group messages by topic
	topicMsgs := make(map[string][]KafkaMessage)
	for _, m := range msgs {
		topicMsgs[m.Topic] = append(topicMsgs[m.Topic], m)
	}

	for topic, mList := range topicMsgs {
		partitions, pErr := conn.GetPartitions(topic)
		if pErr != nil || len(partitions) == 0 {
			partitions = []int{0}
		}

		partMap := make(map[int32][]KafkaMessage)
		for _, m := range mList {
			part := int32(w.Balancer.Balance(m, partitions...))
			partMap[part] = append(partMap[part], m)
		}

		for partID, pMsgs := range partMap {
			if err := conn.ProduceMessage(topic, partID, pMsgs...); err != nil {
				if w.Completion != nil {
					w.Completion(msgs, err)
				}
				return err
			}
		}
	}

	if w.Completion != nil {
		w.Completion(msgs, nil)
	}
	return nil
}

func (w *KafkaWriter) getConn(ctx context.Context) (*KafkaConn, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	broker := w.Brokers[0]
	if conn, exists := w.conns[broker]; exists && conn != nil {
		return conn, nil
	}

	conn, err := DialKafka(ctx, broker, w.Transport)
	if err != nil {
		return nil, err
	}
	if w.conns == nil {
		w.conns = make(map[string]*KafkaConn)
	}
	w.conns[broker] = conn
	return conn, nil
}

// NewKafkaWriterFromConfigRoundRobin creates Kafka writer from Config with RoundRobin balancer
func NewKafkaWriterFromConfigRoundRobin(conf *Config) *KafkaWriter {
	config := DefaultKafkaWriterConfigRoundRobin()

	if conf.App.KafkaModels == "kafka-confluent" {
		kconf := conf.EnsureKafkaConfluent()
		config.Brokers = kconf.BootstrapServers
		config.Transport = createSASLTransport(kconf.ApiKey, kconf.ApiSecret)
	} else {
		config.Brokers = conf.EnsureKafka().Brokers
	}

	return NewKafkaWriter(config)
}

// NewKafkaWriterFromConfigHash creates Kafka writer with Hash balancer from Config
func NewKafkaWriterFromConfigHash(conf *Config) *KafkaWriter {
	config := DefaultKafkaWriterConfigHash()

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
func createSASLTransport(apiKey, apiSecret string) *KafkaTransport {
	return &KafkaTransport{
		TLS: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		SASL: &KafkaSASLPlain{
			Username: apiKey,
			Password: apiSecret,
		},
	}
}

// Close closes all open Kafka connections
func (w *KafkaWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	var firstErr error
	for _, conn := range w.conns {
		if conn != nil {
			if err := conn.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	w.conns = make(map[string]*KafkaConn)
	return firstErr
}

// CloseKafkaWriter safely closes a Kafka writer
func CloseKafkaWriter(writer *KafkaWriter) error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}
