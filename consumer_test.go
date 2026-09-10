package nanopony

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestKafkaConsumerMethods(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers:    []string{"localhost:1"}, // Invalid addr
		Topic:      "test-topic",
		GroupID:    "test-group",
		RetryDelay: 1 * time.Millisecond,
	}

	consumer := NewKafkaConsumer(config)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test ConsumeWithContext - should exit on context timeout or read error
	err := consumer.ConsumeWithContext(ctx, func(message []byte) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error from ConsumeWithContext with invalid Kafka address")
	}

	// Test Close
	if err := consumer.Close(); err != nil {
		t.Errorf("Unexpected error closing consumer: %v", err)
	}
}

func TestKafkaConsumerProto(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers: []string{"localhost:1"},
		Topic:   "test",
	}
	consumer := NewKafkaConsumer(config)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := consumer.ConsumeWithContextProto(ctx,
		func() proto.Message { return &emptypb.Empty{} },
		func(msg proto.Message) error { return nil },
	)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestMessageHandler(t *testing.T) {
	handler := MessageHandler(func(message []byte) error {
		if len(message) == 0 {
			return nil
		}
		return nil
	})

	err := handler([]byte("test message"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestKafkaConsumerConfig(t *testing.T) {
	config := KafkaConsumerConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "test-topic",
		GroupID:     "test-group",
		StartOffset: 0,
	}

	consumer := NewKafkaConsumer(config)
	if consumer == nil {
		t.Fatal("Expected consumer to be created")
	}
	defer consumer.Close()
}

func TestKafkaConsumer_NoBrokers(t *testing.T) {
	consumer := NewKafkaConsumer(KafkaConsumerConfig{})
	err := consumer.ConsumeWithContext(context.Background(), func(b []byte) error { return nil })
	if err == nil {
		t.Error("expected error when no brokers configured")
	}
}

func TestKafkaConsumer_MockConsume(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)

		if apiKey == apiKeyListOffsets {
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "mock-consume-topic")
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0))
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int64(10))
			return buf.Bytes()
		}

		if apiKey == apiKeyFetch {
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "mock-consume-topic")
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0))
			_ = binary.Write(buf, binary.BigEndian, int64(20))

			innerMsg := bytes.NewBuffer(nil)
			innerMsg.WriteByte(1)
			innerMsg.WriteByte(0)
			_ = binary.Write(innerMsg, binary.BigEndian, int64(12345))
			writeKafkaBytes(innerMsg, []byte("key"))
			writeKafkaBytes(innerMsg, []byte("hello-consumed"))
			crc := crc32.ChecksumIEEE(innerMsg.Bytes())

			msgSet := bytes.NewBuffer(nil)
			_ = binary.Write(msgSet, binary.BigEndian, int64(10))
			_ = binary.Write(msgSet, binary.BigEndian, int32(4+innerMsg.Len()))
			_ = binary.Write(msgSet, binary.BigEndian, crc)
			msgSet.Write(innerMsg.Bytes())

			_ = binary.Write(buf, binary.BigEndian, int32(msgSet.Len()))
			buf.Write(msgSet.Bytes())
			return buf.Bytes()
		}

		return nil
	})
	defer cleanup()

	consumer := NewKafkaConsumer(KafkaConsumerConfig{
		Brokers:     []string{addr},
		Topic:       "mock-consume-topic",
		GroupID:     "mock-group",
		StartOffset: LastOffset,
		RetryDelay:  10 * time.Millisecond,
	})
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var receivedMessage atomic.Value

	go func() {
		_ = consumer.ConsumeWithContext(ctx, func(message []byte) error {
			receivedMessage.Store(string(message))
			cancel() // cancel after reading message
			return nil
		})
	}()

	// Wait for message
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if val := receivedMessage.Load(); val != nil {
			if val.(string) == "hello-consumed" {
				return // Success
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("did not receive expected message in time, got: %v", receivedMessage.Load())
}

func TestKafkaConsumer_MockConsumeProto(t *testing.T) {
	protoMsg := &emptypb.Empty{}
	protoBytes, _ := proto.Marshal(protoMsg)

	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		if apiKey == apiKeyFetch {
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "proto-topic")
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0))
			_ = binary.Write(buf, binary.BigEndian, int64(10))

			innerMsg := bytes.NewBuffer(nil)
			innerMsg.WriteByte(1)
			innerMsg.WriteByte(0)
			_ = binary.Write(innerMsg, binary.BigEndian, int64(12345))
			writeKafkaBytes(innerMsg, nil)
			writeKafkaBytes(innerMsg, protoBytes)
			crc := crc32.ChecksumIEEE(innerMsg.Bytes())

			msgSet := bytes.NewBuffer(nil)
			_ = binary.Write(msgSet, binary.BigEndian, int64(0))
			_ = binary.Write(msgSet, binary.BigEndian, int32(4+innerMsg.Len()))
			_ = binary.Write(msgSet, binary.BigEndian, crc)
			msgSet.Write(innerMsg.Bytes())

			_ = binary.Write(buf, binary.BigEndian, int32(msgSet.Len()))
			buf.Write(msgSet.Bytes())
			return buf.Bytes()
		}
		return nil
	})
	defer cleanup()

	consumer := NewKafkaConsumer(KafkaConsumerConfig{
		Brokers:     []string{addr},
		Topic:       "proto-topic",
		StartOffset: 0,
	})
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var receivedProto atomic.Bool

	go func() {
		_ = consumer.ConsumeWithContextProto(ctx,
			func() proto.Message { return &emptypb.Empty{} },
			func(m proto.Message) error {
				receivedProto.Store(true)
				cancel()
				return nil
			},
		)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if receivedProto.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("did not receive proto message")
}

func TestKafkaConsumer_HandlerErrorRetry(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		if apiKey == apiKeyFetch {
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			writeKafkaString(buf, "retry-topic")
			_ = binary.Write(buf, binary.BigEndian, int32(1))
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			_ = binary.Write(buf, binary.BigEndian, int16(0))
			_ = binary.Write(buf, binary.BigEndian, int64(10))

			innerMsg := bytes.NewBuffer(nil)
			innerMsg.WriteByte(1)
			innerMsg.WriteByte(0)
			_ = binary.Write(innerMsg, binary.BigEndian, int64(12345))
			writeKafkaBytes(innerMsg, nil)
			writeKafkaBytes(innerMsg, []byte("retry-data"))
			crc := crc32.ChecksumIEEE(innerMsg.Bytes())

			msgSet := bytes.NewBuffer(nil)
			_ = binary.Write(msgSet, binary.BigEndian, int64(0))
			_ = binary.Write(msgSet, binary.BigEndian, int32(4+innerMsg.Len()))
			_ = binary.Write(msgSet, binary.BigEndian, crc)
			msgSet.Write(innerMsg.Bytes())

			_ = binary.Write(buf, binary.BigEndian, int32(msgSet.Len()))
			buf.Write(msgSet.Bytes())
			return buf.Bytes()
		}
		return nil
	})
	defer cleanup()

	consumer := NewKafkaConsumer(KafkaConsumerConfig{
		Brokers:     []string{addr},
		Topic:       "retry-topic",
		StartOffset: 0,
		RetryDelay:  10 * time.Millisecond,
	})
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var attempts atomic.Int32
	_ = consumer.ConsumeWithContext(ctx, func(b []byte) error {
		attempts.Add(1)
		return errors.New("handler deliberate error")
	})

	if attempts.Load() < 1 {
		t.Errorf("expected at least 1 attempt, got %d", attempts.Load())
	}
}
