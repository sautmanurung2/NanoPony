package nanopony

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Kafka API Keys
const (
	apiKeyProduce      int16 = 0
	apiKeyFetch        int16 = 1
	apiKeyListOffsets  int16 = 2
	apiKeyMetadata     int16 = 3
	apiKeySaslHandshake int16 = 17
	apiKeySaslAuth     int16 = 36
)

// Offset markers
const (
	FirstOffset int64 = -2
	LastOffset  int64 = -1
)

// KafkaError represents an error returned by Kafka broker
type KafkaError struct {
	Code    int16
	Message string
}

func (e *KafkaError) Error() string {
	return fmt.Sprintf("kafka error code %d: %s", e.Code, e.Message)
}

func kafkaErrorCodeToMsg(code int16) string {
	switch code {
	case 0:
		return "NO_ERROR"
	case 1:
		return "OFFSET_OUT_OF_RANGE"
	case 2:
		return "CORRUPT_MESSAGE"
	case 3:
		return "UNKNOWN_TOPIC_OR_PARTITION"
	case 5:
		return "LEADER_NOT_AVAILABLE"
	case 6:
		return "NOT_LEADER_OR_FOLLOWER"
	case 7:
		return "REQUEST_TIMED_OUT"
	case 33:
		return "TOPIC_AUTHORIZATION_FAILED"
	case 35:
		return "CLUSTER_AUTHORIZATION_FAILED"
	case 36:
		return "INVALID_TIMESTAMP"
	case 58:
		return "SASL_AUTHENTICATION_FAILED"
	default:
		return fmt.Sprintf("ERROR_CODE_%d", code)
	}
}

// KafkaConn represents a raw TCP / TLS connection to a Kafka broker
type KafkaConn struct {
	conn      net.Conn
	clientID  string
	corrID    atomic.Int32
	mu        sync.Mutex
	timeout   time.Duration
	tlsConfig *tls.Config
}

// DialKafka dials a Kafka broker address and optionally authenticates with SASL PLAIN
func DialKafka(ctx context.Context, address string, transport *KafkaTransport) (*KafkaConn, error) {
	timeout := 10 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = 100 * time.Millisecond
		}
	}

	var rawConn net.Conn
	var err error

	dialer := &net.Dialer{Timeout: timeout}

	if transport != nil && transport.TLS != nil {
		host, _, splitErr := net.SplitHostPort(address)
		tlsCfg := transport.TLS.Clone()
		if splitErr == nil && tlsCfg.ServerName == "" {
			tlsCfg.ServerName = host
		}
		rawConn, err = tls.DialWithDialer(dialer, "tcp", address, tlsCfg)
	} else {
		rawConn, err = dialer.DialContext(ctx, "tcp", address)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to dial kafka broker %s: %w", address, err)
	}

	kc := &KafkaConn{
		conn:     rawConn,
		clientID: "nanopony",
		timeout:  timeout,
	}

	// If SASL PLAIN credentials are provided, authenticate
	if transport != nil && transport.SASL != nil && transport.SASL.Username != "" {
		if err := kc.authenticateSASLPlain(transport.SASL.Username, transport.SASL.Password); err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("sasl authentication failed: %w", err)
		}
	}

	return kc, nil
}

// authenticateSASLPlain handles SASL PLAIN handshake and authentication
func (kc *KafkaConn) authenticateSASLPlain(username, password string) error {
	// 1. SaslHandshakeRequest (API Key 17, Version 1)
	req := bytes.NewBuffer(nil)
	writeKafkaString(req, "PLAIN")

	resp, err := kc.doRequest(apiKeySaslHandshake, 1, req.Bytes())
	if err != nil {
		return fmt.Errorf("sasl handshake request failed: %w", err)
	}

	if len(resp) < 2 {
		return errors.New("invalid sasl handshake response")
	}
	errCode := int16(binary.BigEndian.Uint16(resp[:2]))
	if errCode != 0 {
		return &KafkaError{Code: errCode, Message: kafkaErrorCodeToMsg(errCode)}
	}

	// 2. Send PLAIN token: \x00<username>\x00<password>
	token := []byte("\x00" + username + "\x00" + password)

	// Try SaslAuthenticateRequest (API Key 36, Version 0)
	authReq := bytes.NewBuffer(nil)
	writeKafkaBytes(authReq, token)

	authResp, err := kc.doRequest(apiKeySaslAuth, 0, authReq.Bytes())
	if err == nil && len(authResp) >= 2 {
		authErrCode := int16(binary.BigEndian.Uint16(authResp[:2]))
		if authErrCode != 0 {
			return &KafkaError{Code: authErrCode, Message: kafkaErrorCodeToMsg(authErrCode)}
		}
		return nil
	}

	// Fallback for brokers accepting raw SASL packet framing directly
	packet := make([]byte, 4+len(token))
	binary.BigEndian.PutUint32(packet[:4], uint32(len(token)))
	copy(packet[4:], token)

	_ = kc.conn.SetDeadline(time.Now().Add(kc.timeout))
	if _, err := kc.conn.Write(packet); err != nil {
		return fmt.Errorf("failed to write raw sasl token: %w", err)
	}

	var respLen uint32
	if err := binary.Read(kc.conn, binary.BigEndian, &respLen); err != nil {
		return fmt.Errorf("failed to read raw sasl auth response length: %w", err)
	}

	authResult := make([]byte, respLen)
	if _, err := io.ReadFull(kc.conn, authResult); err != nil {
		return fmt.Errorf("failed to read raw sasl auth response: %w", err)
	}

	return nil
}

// doRequest sends a Kafka wire request and reads the response
func (kc *KafkaConn) doRequest(apiKey, apiVersion int16, body []byte) ([]byte, error) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	corrID := kc.corrID.Add(1)

	// Build Request Header v1:
	// apiKey (int16), apiVersion (int16), correlationId (int32), clientId (string)
	headerBuf := bytes.NewBuffer(nil)
	_ = binary.Write(headerBuf, binary.BigEndian, apiKey)
	_ = binary.Write(headerBuf, binary.BigEndian, apiVersion)
	_ = binary.Write(headerBuf, binary.BigEndian, corrID)
	writeKafkaString(headerBuf, kc.clientID)

	reqPayload := append(headerBuf.Bytes(), body...)
	totalLen := int32(len(reqPayload))

	_ = kc.conn.SetDeadline(time.Now().Add(kc.timeout))

	// Write total size (4 bytes) + payload
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(totalLen))
	if _, err := kc.conn.Write(lenBuf[:]); err != nil {
		return nil, fmt.Errorf("failed to write request size: %w", err)
	}
	if _, err := kc.conn.Write(reqPayload); err != nil {
		return nil, fmt.Errorf("failed to write request body: %w", err)
	}

	// Read response total size (4 bytes)
	if _, err := io.ReadFull(kc.conn, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("failed to read response size: %w", err)
	}
	respSize := binary.BigEndian.Uint32(lenBuf[:])
	if respSize < 4 {
		return nil, fmt.Errorf("invalid response size: %d", respSize)
	}

	respData := make([]byte, respSize)
	if _, err := io.ReadFull(kc.conn, respData); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respCorrID := int32(binary.BigEndian.Uint32(respData[:4]))
	if respCorrID != corrID {
		return nil, fmt.Errorf("correlation ID mismatch: expected %d, got %d", corrID, respCorrID)
	}

	return respData[4:], nil
}

// GetPartitions queries topic metadata to get available partition IDs
func (kc *KafkaConn) GetPartitions(topic string) ([]int, error) {
	req := bytes.NewBuffer(nil)
	// Topics array (int32 count + strings)
	_ = binary.Write(req, binary.BigEndian, int32(1))
	writeKafkaString(req, topic)

	resp, err := kc.doRequest(apiKeyMetadata, 1, req.Bytes())
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(resp)

	// Read brokers count
	var brokerCount int32
	if err := binary.Read(r, binary.BigEndian, &brokerCount); err != nil {
		return nil, err
	}
	for i := 0; i < int(brokerCount); i++ {
		var nodeID int32
		_ = binary.Read(r, binary.BigEndian, &nodeID)
		_ = readKafkaString(r) // host
		var port int32
		_ = binary.Read(r, binary.BigEndian, &port)
		_ = readKafkaString(r) // rack
	}

	var controllerID int32
	_ = binary.Read(r, binary.BigEndian, &controllerID)

	var topicCount int32
	if err := binary.Read(r, binary.BigEndian, &topicCount); err != nil {
		return nil, err
	}

	var partitions []int
	for i := 0; i < int(topicCount); i++ {
		var errCode int16
		_ = binary.Read(r, binary.BigEndian, &errCode)
		tName := readKafkaString(r)
		var isInternal bool
		_ = binary.Read(r, binary.BigEndian, &isInternal)

		var partCount int32
		_ = binary.Read(r, binary.BigEndian, &partCount)
		for j := 0; j < int(partCount); j++ {
			var pErr int16
			var partID, leader int32
			_ = binary.Read(r, binary.BigEndian, &pErr)
			_ = binary.Read(r, binary.BigEndian, &partID)
			_ = binary.Read(r, binary.BigEndian, &leader)

			// read replicas
			var repCount int32
			_ = binary.Read(r, binary.BigEndian, &repCount)
			for k := 0; k < int(repCount); k++ {
				var rep int32
				_ = binary.Read(r, binary.BigEndian, &rep)
			}
			// read isr
			var isrCount int32
			_ = binary.Read(r, binary.BigEndian, &isrCount)
			for k := 0; k < int(isrCount); k++ {
				var isr int32
				_ = binary.Read(r, binary.BigEndian, &isr)
			}

			if tName == topic {
				partitions = append(partitions, int(partID))
			}
		}

		if tName == topic && errCode != 0 {
			return nil, &KafkaError{Code: errCode, Message: kafkaErrorCodeToMsg(errCode)}
		}
	}

	if len(partitions) == 0 {
		return []int{0}, nil
	}
	return partitions, nil
}

// ProduceMessage sends a batch of messages to a topic partition using ProduceRequest v2
func (kc *KafkaConn) ProduceMessage(topic string, partition int32, msgs ...KafkaMessage) error {
	// Build MessageSet
	msgSetBuf := bytes.NewBuffer(nil)
	now := time.Now().UnixMilli()

	for _, m := range msgs {
		msgBuf := bytes.NewBuffer(nil)
		// Magic: 1
		msgBuf.WriteByte(1)
		// Attributes: 0
		msgBuf.WriteByte(0)
		// Timestamp: int64
		_ = binary.Write(msgBuf, binary.BigEndian, now)
		// Key bytes
		writeKafkaBytes(msgBuf, m.Key)
		// Value bytes
		writeKafkaBytes(msgBuf, m.Value)

		// Calculate CRC32 of msgBuf
		crc := crc32.ChecksumIEEE(msgBuf.Bytes())

		// Write to MessageSet:
		// Offset (int64) = 0
		_ = binary.Write(msgSetBuf, binary.BigEndian, int64(0))
		// MessageSize (int32) = 4 (crc) + len(msgBuf)
		_ = binary.Write(msgSetBuf, binary.BigEndian, int32(4+msgBuf.Len()))
		// CRC (uint32)
		_ = binary.Write(msgSetBuf, binary.BigEndian, crc)
		// Message payload
		msgSetBuf.Write(msgBuf.Bytes())
	}

	// Build ProduceRequest v2:
	// acks (int16), timeout (int32), topics array
	req := bytes.NewBuffer(nil)
	_ = binary.Write(req, binary.BigEndian, int16(1))     // RequiredAcks = 1
	_ = binary.Write(req, binary.BigEndian, int32(10000)) // Timeout = 10s

	// Topics array count = 1
	_ = binary.Write(req, binary.BigEndian, int32(1))
	writeKafkaString(req, topic)

	// Partitions array count = 1
	_ = binary.Write(req, binary.BigEndian, int32(1))
	_ = binary.Write(req, binary.BigEndian, partition)
	// MessageSet size & data
	_ = binary.Write(req, binary.BigEndian, int32(msgSetBuf.Len()))
	req.Write(msgSetBuf.Bytes())

	resp, err := kc.doRequest(apiKeyProduce, 2, req.Bytes())
	if err != nil {
		return err
	}

	// Parse ProduceResponse v2:
	// topics array
	r := bytes.NewReader(resp)
	var topicCount int32
	if err := binary.Read(r, binary.BigEndian, &topicCount); err != nil {
		return err
	}
	for i := 0; i < int(topicCount); i++ {
		_ = readKafkaString(r)
		var partCount int32
		_ = binary.Read(r, binary.BigEndian, &partCount)
		for j := 0; j < int(partCount); j++ {
			var p int32
			var errCode int16
			var baseOffset, logAppendTime int64
			_ = binary.Read(r, binary.BigEndian, &p)
			_ = binary.Read(r, binary.BigEndian, &errCode)
			_ = binary.Read(r, binary.BigEndian, &baseOffset)
			_ = binary.Read(r, binary.BigEndian, &logAppendTime)

			if errCode != 0 {
				return &KafkaError{Code: errCode, Message: kafkaErrorCodeToMsg(errCode)}
			}
		}
	}

	return nil
}

// FetchMessages fetches messages from topic and partition starting at offset
func (kc *KafkaConn) FetchMessages(topic string, partition int32, offset int64, maxBytes int32) ([]KafkaMessage, int64, error) {
	req := bytes.NewBuffer(nil)
	// ReplicaID: -1
	_ = binary.Write(req, binary.BigEndian, int32(-1))
	// MaxWaitTime: 500ms
	_ = binary.Write(req, binary.BigEndian, int32(500))
	// MinBytes: 1
	_ = binary.Write(req, binary.BigEndian, int32(1))

	// Topics count: 1
	_ = binary.Write(req, binary.BigEndian, int32(1))
	writeKafkaString(req, topic)

	// Partitions count: 1
	_ = binary.Write(req, binary.BigEndian, int32(1))
	_ = binary.Write(req, binary.BigEndian, partition)
	_ = binary.Write(req, binary.BigEndian, offset)
	_ = binary.Write(req, binary.BigEndian, maxBytes)

	resp, err := kc.doRequest(apiKeyFetch, 2, req.Bytes())
	if err != nil {
		return nil, offset, err
	}

	r := bytes.NewReader(resp)
	var throttleTime int32
	_ = binary.Read(r, binary.BigEndian, &throttleTime)

	var topicCount int32
	if err := binary.Read(r, binary.BigEndian, &topicCount); err != nil {
		return nil, offset, err
	}

	var messages []KafkaMessage
	nextOffset := offset

	for i := 0; i < int(topicCount); i++ {
		tName := readKafkaString(r)
		var partCount int32
		_ = binary.Read(r, binary.BigEndian, &partCount)
		for j := 0; j < int(partCount); j++ {
			var p int32
			var errCode int16
			var highWatermark int64
			var msgSetSize int32

			_ = binary.Read(r, binary.BigEndian, &p)
			_ = binary.Read(r, binary.BigEndian, &errCode)
			_ = binary.Read(r, binary.BigEndian, &highWatermark)
			_ = binary.Read(r, binary.BigEndian, &msgSetSize)

			if errCode != 0 {
				return nil, offset, &KafkaError{Code: errCode, Message: kafkaErrorCodeToMsg(errCode)}
			}

			// Read MessageSet bytes
			if msgSetSize <= 0 {
				continue
			}
			msgSetBytes := make([]byte, msgSetSize)
			if _, err := io.ReadFull(r, msgSetBytes); err != nil {
				continue
			}

			setReader := bytes.NewReader(msgSetBytes)
			for setReader.Len() > 0 {
				var msgOffset int64
				var msgSize int32
				if err := binary.Read(setReader, binary.BigEndian, &msgOffset); err != nil {
					break
				}
				if err := binary.Read(setReader, binary.BigEndian, &msgSize); err != nil {
					break
				}
				if msgSize <= 0 || int(msgSize) > setReader.Len() {
					break
				}

				singleMsg := make([]byte, msgSize)
				if _, err := io.ReadFull(setReader, singleMsg); err != nil {
					break
				}

				// singleMsg has: crc (4), magic (1), attributes (1), timestamp (8), key, value
				if len(singleMsg) < 14 {
					continue
				}

				smReader := bytes.NewReader(singleMsg[14:])
				key := readKafkaBytes(smReader)
				val := readKafkaBytes(smReader)

				messages = append(messages, KafkaMessage{
					Topic:     tName,
					Partition: int(p),
					Offset:    msgOffset,
					Key:       key,
					Value:     val,
				})
				nextOffset = msgOffset + 1
			}
		}
	}

	return messages, nextOffset, nil
}

// GetOffset queries broker for earliest or latest offset
func (kc *KafkaConn) GetOffset(topic string, partition int32, timeMarker int64) (int64, error) {
	req := bytes.NewBuffer(nil)
	_ = binary.Write(req, binary.BigEndian, int32(-1)) // ReplicaID: -1
	_ = binary.Write(req, binary.BigEndian, int32(1))  // Topics count: 1
	writeKafkaString(req, topic)
	_ = binary.Write(req, binary.BigEndian, int32(1)) // Partitions count: 1
	_ = binary.Write(req, binary.BigEndian, partition)
	_ = binary.Write(req, binary.BigEndian, timeMarker)

	resp, err := kc.doRequest(apiKeyListOffsets, 1, req.Bytes())
	if err != nil {
		return 0, err
	}

	r := bytes.NewReader(resp)
	var topicCount int32
	if err := binary.Read(r, binary.BigEndian, &topicCount); err != nil {
		return 0, err
	}
	for i := 0; i < int(topicCount); i++ {
		_ = readKafkaString(r)
		var partCount int32
		_ = binary.Read(r, binary.BigEndian, &partCount)
		for j := 0; j < int(partCount); j++ {
			var p int32
			var errCode int16
			var offCount int32
			_ = binary.Read(r, binary.BigEndian, &p)
			_ = binary.Read(r, binary.BigEndian, &errCode)
			_ = binary.Read(r, binary.BigEndian, &offCount)

			if errCode != 0 {
				return 0, &KafkaError{Code: errCode, Message: kafkaErrorCodeToMsg(errCode)}
			}
			if offCount > 0 {
				var offset int64
				_ = binary.Read(r, binary.BigEndian, &offset)
				return offset, nil
			}
		}
	}

	return 0, nil
}

// Close closes the underlying network connection
func (kc *KafkaConn) Close() error {
	if kc != nil && kc.conn != nil {
		return kc.conn.Close()
	}
	return nil
}

// Binary serialization helpers
func writeKafkaString(buf *bytes.Buffer, s string) {
	if s == "" {
		_ = binary.Write(buf, binary.BigEndian, int16(0))
		return
	}
	_ = binary.Write(buf, binary.BigEndian, int16(len(s)))
	buf.WriteString(s)
}

func readKafkaString(r io.Reader) string {
	var length int16
	if err := binary.Read(r, binary.BigEndian, &length); err != nil || length <= 0 {
		return ""
	}
	data := make([]byte, length)
	_, _ = io.ReadFull(r, data)
	return string(data)
}

func writeKafkaBytes(buf *bytes.Buffer, b []byte) {
	if b == nil {
		_ = binary.Write(buf, binary.BigEndian, int32(-1))
		return
	}
	_ = binary.Write(buf, binary.BigEndian, int32(len(b)))
	buf.Write(b)
}

func readKafkaBytes(r io.Reader) []byte {
	var length int32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil || length < 0 {
		return nil
	}
	if length == 0 {
		return []byte{}
	}
	data := make([]byte, length)
	_, _ = io.ReadFull(r, data)
	return data
}
