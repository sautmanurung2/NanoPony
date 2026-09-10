package nanopony

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
)

func TestNewKafkaWriter(t *testing.T) {
	config := KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
	}

	writer := NewKafkaWriter(config)
	if writer == nil {
		t.Fatal("Expected Kafka writer to be created")
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewKafkaWriterFromConfigRoundRobin(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	// This will create a writer, but may fail if Kafka is not available
	writer := NewKafkaWriterFromConfigRoundRobin(config)
	if writer == nil {
		t.Log("Warning: Writer not created (Kafka may not be configured)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCloseKafkaWriter(t *testing.T) {
	// Test closing nil writer
	err := CloseKafkaWriter(nil)
	if err != nil {
		t.Errorf("Expected no error when closing nil writer, got %v", err)
	}

	// Test closing actual writer
	config := KafkaWriterConfig{
		Brokers: []string{"localhost:9092"},
	}
	writer := NewKafkaWriter(config)

	err = CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test closing already closed writer (should handle gracefully)
	err = CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Expected no error when closing already closed writer, got %v", err)
	}
}

func TestKafkaWriterConfig(t *testing.T) {
	config := KafkaWriterConfig{
		Brokers:      []string{"broker1:9092", "broker2:9092"},
		Balancer:     &RoundRobin{},
		BatchTimeout: 0,
		Transport:    nil,
	}

	if len(config.Brokers) != 2 {
		t.Errorf("Expected 2 brokers, got %d", len(config.Brokers))
	}
	if config.Balancer == nil {
		t.Error("Expected balancer to be set")
	}
}

func TestCreateSASLTransport(t *testing.T) {
	// Test creating SASL transport for Confluent Cloud
	apiKey := "test-api-key"
	apiSecret := "test-api-secret"

	transport := createSASLTransport(apiKey, apiSecret)
	if transport == nil {
		t.Fatal("Expected SASL transport to be created")
	}
	if transport.SASL == nil || transport.SASL.Username != apiKey || transport.SASL.Password != apiSecret {
		t.Fatal("Expected SASL credentials to match")
	}
}

func TestNewKafkaWriterFromConfluentConfigRoundRobin(t *testing.T) {
	ResetConfig()

	// Manually set confluent config
	conf := &Config{}
	conf.App.KafkaModels = "kafka-confluent"
	conf.KafkaConfluent.ApiKey = "test-key"
	conf.KafkaConfluent.ApiSecret = "test-secret"
	conf.KafkaConfluent.BootstrapServers = []string{"pkc-test.us-east-1.aws.confluent.cloud:9092"}

	// This should create a writer with SASL transport
	writer := NewKafkaWriterFromConfigRoundRobin(conf)
	if writer == nil {
		t.Log("Warning: Writer not created (may be expected)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewKafkaWriterFromConfigHash(t *testing.T) {
	ResetConfig()
	config := NewConfig()

	writer := NewKafkaWriterFromConfigHash(config)
	if writer == nil {
		t.Log("Warning: Writer not created (Kafka may not be configured)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewKafkaWriterFromConfluentConfigHash(t *testing.T) {
	ResetConfig()

	// Manually set confluent config
	conf := &Config{}
	conf.App.KafkaModels = "kafka-confluent"
	conf.KafkaConfluent.ApiKey = "test-key"
	conf.KafkaConfluent.ApiSecret = "test-secret"
	conf.KafkaConfluent.BootstrapServers = []string{"pkc-test.us-east-1.aws.confluent.cloud:9092"}

	// This should create a writer with SASL transport
	writer := NewKafkaWriterFromConfigHash(conf)
	if writer == nil {
		t.Log("Warning: Writer not created (may be expected)")
		return
	}

	// Clean up
	err := CloseKafkaWriter(writer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestBalancers(t *testing.T) {
	rr := &RoundRobin{}
	p0 := rr.Balance(KafkaMessage{}, 0, 1, 2)
	p1 := rr.Balance(KafkaMessage{}, 0, 1, 2)
	p2 := rr.Balance(KafkaMessage{}, 0, 1, 2)
	if p0 != 0 || p1 != 1 || p2 != 2 {
		t.Errorf("RoundRobin balance mismatch: got %d, %d, %d", p0, p1, p2)
	}

	// Edge case: empty partitions
	pEmpty := rr.Balance(KafkaMessage{})
	if pEmpty != 0 {
		t.Errorf("Expected 0 for empty partitions, got %d", pEmpty)
	}

	h := &Hash{}
	hp := h.Balance(KafkaMessage{Key: []byte("test-key")}, 0, 1, 2)
	if hp < 0 || hp > 2 {
		t.Errorf("Hash balance out of range: %d", hp)
	}

	// Edge cases for Hash
	if h.Balance(KafkaMessage{}) != 0 {
		t.Errorf("Expected 0 for empty partitions in Hash")
	}
	if h.Balance(KafkaMessage{Key: nil}, 0, 1) != 0 {
		t.Errorf("Expected partitions[0] for nil key in Hash")
	}
}

func TestKafkaWriter_WriteMessages_EdgeCases(t *testing.T) {
	var nilWriter *KafkaWriter
	if err := nilWriter.WriteMessages(context.Background()); err == nil {
		t.Error("expected error for nil writer")
	}

	emptyWriter := &KafkaWriter{}
	if err := emptyWriter.WriteMessages(context.Background()); err == nil {
		t.Error("expected error for writer with no brokers")
	}
}

func TestKafkaWriter_WriteMessages_MockServer(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		if apiKey == apiKeyMetadata {
			// Metadata: 1 broker, 1 topic with 2 partitions
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "127.0.0.1")
			_ = binary.Write(buf, binary.BigEndian, int32(9092))
			writeKafkaString(buf, "")
			_ = binary.Write(buf, binary.BigEndian, int32(1)) // controller_id

			_ = binary.Write(buf, binary.BigEndian, int32(1)) // topic count
			_ = binary.Write(buf, binary.BigEndian, int16(0)) // err 0
			writeKafkaString(buf, "mock-topic")
			buf.WriteByte(0)
			_ = binary.Write(buf, binary.BigEndian, int32(2)) // 2 partitions
			_ = binary.Write(buf, binary.BigEndian, int16(0)) // p0 err
			_ = binary.Write(buf, binary.BigEndian, int32(0)) // p0 id
			_ = binary.Write(buf, binary.BigEndian, int32(1)) // p0 leader
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0)) // p1 err
			_ = binary.Write(buf, binary.BigEndian, int32(1)) // p1 id
			_ = binary.Write(buf, binary.BigEndian, int32(1)) // p1 leader
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			return buf.Bytes()
		}

		if apiKey == apiKeyProduce {
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "mock-topic")
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0))
			_ = binary.Write(buf, binary.BigEndian, int64(1))
			_ = binary.Write(buf, binary.BigEndian, int64(0))
			return buf.Bytes()
		}

		return nil
	})
	defer cleanup()

	w := NewKafkaWriter(KafkaWriterConfig{
		Brokers: []string{addr},
	})
	defer w.Close()

	logger := NewLoggerFromOptions(LoggerOptions{ServiceName: "test-writer"})

	msgs := []KafkaMessage{
		{
			Topic: "mock-topic",
			Key:   []byte("key1"),
			Value: []byte("val1"),
			WriterData: KafkaMessageMetadata{
				LoggerEntry: logger,
				Payload:     "payload-data",
				LogData:     "log-info",
			},
		},
		{
			Topic: "mock-topic",
			Key:   []byte("key2"),
			Value: []byte("val2"),
		},
	}

	err := w.WriteMessages(context.Background(), msgs...)
	if err != nil {
		t.Fatalf("unexpected WriteMessages error: %v", err)
	}

	// Verify getConn caches the connection
	conn1, err := w.getConn(context.Background())
	if err != nil {
		t.Fatalf("getConn error: %v", err)
	}
	conn2, err := w.getConn(context.Background())
	if err != nil {
		t.Fatalf("getConn error: %v", err)
	}
	if conn1 != conn2 {
		t.Errorf("expected cached connection, got different pointers")
	}
}
