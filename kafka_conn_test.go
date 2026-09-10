package nanopony

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"net"
	"testing"
)

// helper to start a mock TCP Kafka broker
func startMockKafkaServer(t *testing.T, handler func(apiKey, apiVersion int16, corrID int32, body []byte) []byte) (string, func()) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on local port: %v", err)
	}

	stopChan := make(chan struct{})

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-stopChan:
					return
				default:
					return
				}
			}

			go func(c net.Conn) {
				defer c.Close()
				for {
					var size uint32
					if err := binary.Read(c, binary.BigEndian, &size); err != nil {
						return
					}
					reqBytes := make([]byte, size)
					if _, err := io.ReadFull(c, reqBytes); err != nil {
						return
					}

					r := bytes.NewReader(reqBytes)
					var apiKey, apiVersion int16
					var corrID int32
					_ = binary.Read(r, binary.BigEndian, &apiKey)
					_ = binary.Read(r, binary.BigEndian, &apiVersion)
					_ = binary.Read(r, binary.BigEndian, &corrID)
					_ = readKafkaString(r) // clientID

					body := make([]byte, r.Len())
					_, _ = r.Read(body)

					respBody := handler(apiKey, apiVersion, corrID, body)

					// Frame response: [size][corrID][respBody]
					respBuf := bytes.NewBuffer(nil)
					_ = binary.Write(respBuf, binary.BigEndian, corrID)
					respBuf.Write(respBody)

					totalSize := uint32(respBuf.Len())
					_ = binary.Write(c, binary.BigEndian, totalSize)
					_, _ = c.Write(respBuf.Bytes())
				}
			}(conn)
		}
	}()

	cleanup := func() {
		close(stopChan)
		_ = l.Close()
	}

	return l.Addr().String(), cleanup
}

func TestKafkaConn_Metadata(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		// brokers count = 1
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int32(1)) // node_id
		writeKafkaString(buf, "127.0.0.1")
		_ = binary.Write(buf, binary.BigEndian, int32(9092))
		writeKafkaString(buf, "")

		_ = binary.Write(buf, binary.BigEndian, int32(1)) // controller_id

		// topics count = 1
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int16(0)) // errCode
		writeKafkaString(buf, "test-topic")
		buf.WriteByte(0) // is_internal

		// partitions count = 2
		_ = binary.Write(buf, binary.BigEndian, int32(2))
		// p0
		_ = binary.Write(buf, binary.BigEndian, int16(0))
		_ = binary.Write(buf, binary.BigEndian, int32(0))
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // replicas
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // isr
		// p1
		_ = binary.Write(buf, binary.BigEndian, int16(0))
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int32(0))
		_ = binary.Write(buf, binary.BigEndian, int32(0))

		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	parts, err := conn.GetPartitions("test-topic")
	if err != nil {
		t.Fatalf("failed to get partitions: %v", err)
	}
	if len(parts) != 2 || parts[0] != 0 || parts[1] != 1 {
		t.Errorf("unexpected partitions: %v", parts)
	}
}

func TestKafkaConn_MetadataError(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // brokers
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // controller
		_ = binary.Write(buf, binary.BigEndian, int32(1)) // topics count
		_ = binary.Write(buf, binary.BigEndian, int16(3)) // errCode UNKNOWN_TOPIC_OR_PARTITION
		writeKafkaString(buf, "unknown-topic")
		buf.WriteByte(0)
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // partitions count
		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	_, err = conn.GetPartitions("unknown-topic")
	if err == nil {
		t.Error("expected error for unknown topic")
	}
}

func TestKafkaConn_ProduceMessage(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		// ProduceResponse v2
		_ = binary.Write(buf, binary.BigEndian, int32(1)) // topics count
		writeKafkaString(buf, "test-topic")
		_ = binary.Write(buf, binary.BigEndian, int32(1))  // partitions count
		_ = binary.Write(buf, binary.BigEndian, int32(0))  // part 0
		_ = binary.Write(buf, binary.BigEndian, int16(0))  // errCode 0
		_ = binary.Write(buf, binary.BigEndian, int64(42)) // baseOffset
		_ = binary.Write(buf, binary.BigEndian, int64(0))  // logAppendTime
		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	err = conn.ProduceMessage("test-topic", 0, KafkaMessage{
		Key:   []byte("key1"),
		Value: []byte("val1"),
	})
	if err != nil {
		t.Fatalf("unexpected produce error: %v", err)
	}
}

func TestKafkaConn_ProduceMessageError(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		writeKafkaString(buf, "test-topic")
		_ = binary.Write(buf, binary.BigEndian, int32(1))
		_ = binary.Write(buf, binary.BigEndian, int32(0))
		_ = binary.Write(buf, binary.BigEndian, int16(6)) // NOT_LEADER_OR_FOLLOWER
		_ = binary.Write(buf, binary.BigEndian, int64(0))
		_ = binary.Write(buf, binary.BigEndian, int64(0))
		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	err = conn.ProduceMessage("test-topic", 0, KafkaMessage{Value: []byte("val")})
	if err == nil {
		t.Error("expected produce error")
	}
}

func TestKafkaConn_FetchMessages(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		_ = binary.Write(buf, binary.BigEndian, int32(0)) // throttleTime
		_ = binary.Write(buf, binary.BigEndian, int32(1)) // topics count
		writeKafkaString(buf, "test-topic")
		_ = binary.Write(buf, binary.BigEndian, int32(1))   // part count
		_ = binary.Write(buf, binary.BigEndian, int32(0))   // part 0
		_ = binary.Write(buf, binary.BigEndian, int16(0))   // errCode
		_ = binary.Write(buf, binary.BigEndian, int64(100)) // highWatermark

		// Build a single message inside MessageSet
		innerMsg := bytes.NewBuffer(nil)
		innerMsg.WriteByte(1)                                       // magic
		innerMsg.WriteByte(0)                                       // attributes
		_ = binary.Write(innerMsg, binary.BigEndian, int64(123456)) // timestamp
		writeKafkaBytes(innerMsg, []byte("key-fetch"))
		writeKafkaBytes(innerMsg, []byte("val-fetch"))

		crc := crc32.ChecksumIEEE(innerMsg.Bytes())

		msgSet := bytes.NewBuffer(nil)
		_ = binary.Write(msgSet, binary.BigEndian, int64(10))               // offset
		_ = binary.Write(msgSet, binary.BigEndian, int32(4+innerMsg.Len())) // msgSize
		_ = binary.Write(msgSet, binary.BigEndian, crc)
		msgSet.Write(innerMsg.Bytes())

		_ = binary.Write(buf, binary.BigEndian, int32(msgSet.Len())) // msgSetSize
		buf.Write(msgSet.Bytes())

		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	msgs, nextOff, err := conn.FetchMessages("test-topic", 0, 10, 1048576)
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if string(msgs[0].Key) != "key-fetch" || string(msgs[0].Value) != "val-fetch" {
		t.Errorf("unexpected message content: %s / %s", msgs[0].Key, msgs[0].Value)
	}
	if nextOff != 11 {
		t.Errorf("expected nextOffset 11, got %d", nextOff)
	}
}

func TestKafkaConn_GetOffset(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		_ = binary.Write(buf, binary.BigEndian, int32(1)) // topics count
		writeKafkaString(buf, "test-topic")
		_ = binary.Write(buf, binary.BigEndian, int32(1))  // part count
		_ = binary.Write(buf, binary.BigEndian, int32(0))  // part 0
		_ = binary.Write(buf, binary.BigEndian, int16(0))  // errCode 0
		_ = binary.Write(buf, binary.BigEndian, int32(1))  // off count
		_ = binary.Write(buf, binary.BigEndian, int64(88)) // offset
		return buf.Bytes()
	})
	defer cleanup()

	conn, err := DialKafka(context.Background(), addr, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	offset, err := conn.GetOffset("test-topic", 0, LastOffset)
	if err != nil {
		t.Fatalf("unexpected get offset error: %v", err)
	}
	if offset != 88 {
		t.Errorf("expected offset 88, got %d", offset)
	}
}

func TestKafkaConn_SASLPlainAuth(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		if apiKey == apiKeySaslHandshake {
			// SaslHandshakeResponse
			_ = binary.Write(buf, binary.BigEndian, int16(0)) // errCode 0
			_ = binary.Write(buf, binary.BigEndian, int32(1)) // mechanisms count
			writeKafkaString(buf, "PLAIN")
			return buf.Bytes()
		}
		if apiKey == apiKeySaslAuth {
			// SaslAuthResponse
			_ = binary.Write(buf, binary.BigEndian, int16(0)) // errCode 0
			writeKafkaString(buf, "")                         // errMsg
			_ = binary.Write(buf, binary.BigEndian, int32(0)) // bytes len
			return buf.Bytes()
		}
		return nil
	})
	defer cleanup()

	transport := &KafkaTransport{
		SASL: &KafkaSASLPlain{
			Username: "user",
			Password: "pass",
		},
	}

	conn, err := DialKafka(context.Background(), addr, transport)
	if err != nil {
		t.Fatalf("failed to dial with sasl: %v", err)
	}
	_ = conn.Close()
}

func TestKafkaConn_SASLPlainAuthFailed(t *testing.T) {
	addr, cleanup := startMockKafkaServer(t, func(apiKey, apiVersion int16, corrID int32, body []byte) []byte {
		buf := bytes.NewBuffer(nil)
		if apiKey == apiKeySaslHandshake {
			_ = binary.Write(buf, binary.BigEndian, int16(58)) // SASL_AUTHENTICATION_FAILED
			_ = binary.Write(buf, binary.BigEndian, int32(0))
			return buf.Bytes()
		}
		return nil
	})
	defer cleanup()

	transport := &KafkaTransport{
		SASL: &KafkaSASLPlain{
			Username: "bad-user",
			Password: "bad-password",
		},
	}

	_, err := DialKafka(context.Background(), addr, transport)
	if err == nil {
		t.Error("expected auth error")
	}
}

func TestKafkaConn_SerializationHelpers(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	// String tests
	writeKafkaString(buf, "")
	writeKafkaString(buf, "hello")
	s1 := readKafkaString(buf)
	s2 := readKafkaString(buf)
	if s1 != "" || s2 != "hello" {
		t.Errorf("read string mismatch: '%s', '%s'", s1, s2)
	}

	// Bytes tests
	writeKafkaBytes(buf, nil)
	writeKafkaBytes(buf, []byte{})
	writeKafkaBytes(buf, []byte("data"))

	b1 := readKafkaBytes(buf)
	b2 := readKafkaBytes(buf)
	b3 := readKafkaBytes(buf)
	if b1 != nil || len(b2) != 0 || string(b3) != "data" {
		t.Errorf("read bytes mismatch: %v, %v, %v", b1, b2, b3)
	}
}

func TestKafkaConn_ErrorCodes(t *testing.T) {
	codes := []int16{0, 1, 2, 3, 5, 6, 7, 33, 35, 36, 58, 999}
	for _, c := range codes {
		msg := kafkaErrorCodeToMsg(c)
		if msg == "" {
			t.Errorf("empty error message for code %d", c)
		}
		ke := &KafkaError{Code: c, Message: msg}
		if ke.Error() == "" {
			t.Errorf("empty KafkaError string for code %d", c)
		}
	}
}

func TestKafkaConn_CloseNil(t *testing.T) {
	var conn *KafkaConn
	if err := conn.Close(); err != nil {
		t.Errorf("expected nil error on nil Close, got %v", err)
	}
}
