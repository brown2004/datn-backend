package mqtt

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type Message struct {
	Topic   string
	Payload []byte
}

type Handler func(context.Context, Message) error

type Subscriber struct {
	Broker      string
	ClientID    string
	Username    string
	Password    string
	TopicFilter string
	Logger      *log.Logger
}

func (s *Subscriber) Run(ctx context.Context, handler Handler) error {
	if strings.TrimSpace(s.Broker) == "" {
		return fmt.Errorf("mqtt broker is empty")
	}
	if strings.TrimSpace(s.TopicFilter) == "" {
		return fmt.Errorf("mqtt topic filter is empty")
	}
	if handler == nil {
		return fmt.Errorf("mqtt handler is nil")
	}

	for {
		if err := s.runOnce(ctx, handler); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			s.logf("subscriber disconnected: %v", err)
		}

		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (s *Subscriber) runOnce(ctx context.Context, handler Handler) error {
	addr := brokerAddress(s.Broker)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	if err := s.connect(conn); err != nil {
		return err
	}
	if err := s.subscribe(conn); err != nil {
		return err
	}

	s.logf("subscribed broker=%s topic=%s", addr, s.TopicFilter)
	return s.readLoop(ctx, conn, handler)
}

func (s *Subscriber) connect(conn net.Conn) error {
	clientID := strings.TrimSpace(s.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("datn-backend-%d", time.Now().UnixNano())
	}

	var variableHeader bytes.Buffer
	writeUTF(&variableHeader, "MQTT")
	variableHeader.WriteByte(4)

	connectFlags := byte(0x02)
	if s.Username != "" {
		connectFlags |= 0x80
	}
	if s.Password != "" {
		connectFlags |= 0x40
	}
	variableHeader.WriteByte(connectFlags)
	_ = binary.Write(&variableHeader, binary.BigEndian, uint16(0))

	var payload bytes.Buffer
	writeUTF(&payload, clientID)
	if s.Username != "" {
		writeUTF(&payload, s.Username)
	}
	if s.Password != "" {
		writeUTF(&payload, s.Password)
	}

	packetPayload := append(variableHeader.Bytes(), payload.Bytes()...)
	if err := writePacket(conn, 1, 0, packetPayload); err != nil {
		return err
	}

	packetType, _, response, err := readPacket(conn)
	if err != nil {
		return err
	}
	if packetType != 2 || len(response) < 2 {
		return fmt.Errorf("invalid mqtt connack")
	}
	if response[1] != 0 {
		return fmt.Errorf("mqtt connack refused code=%d", response[1])
	}

	return nil
}

func (s *Subscriber) subscribe(conn net.Conn) error {
	var payload bytes.Buffer
	_ = binary.Write(&payload, binary.BigEndian, uint16(1))
	writeUTF(&payload, s.TopicFilter)
	payload.WriteByte(1)

	if err := writePacket(conn, 8, 2, payload.Bytes()); err != nil {
		return err
	}

	packetType, _, response, err := readPacket(conn)
	if err != nil {
		return err
	}
	if packetType != 9 || len(response) < 3 {
		return fmt.Errorf("invalid mqtt suback")
	}
	if response[2] == 0x80 {
		return fmt.Errorf("mqtt subscribe refused topic=%s", s.TopicFilter)
	}

	return nil
}

func (s *Subscriber) readLoop(ctx context.Context, conn net.Conn, handler Handler) error {
	for {
		packetType, flags, payload, err := readPacket(conn)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		if packetType != 3 {
			continue
		}

		topic, packetID, messagePayload, err := parsePublish(flags, payload)
		if err != nil {
			s.logf("skip invalid publish: %v", err)
			continue
		}

		if err := handler(ctx, Message{Topic: topic, Payload: messagePayload}); err != nil {
			s.logf("handle publish failed topic=%s: %v", topic, err)
		}

		if len(packetID) == 2 {
			if err := writePacket(conn, 4, 0, packetID); err != nil {
				return err
			}
		}
	}
}

func parsePublish(flags byte, payload []byte) (string, []byte, []byte, error) {
	if len(payload) < 2 {
		return "", nil, nil, fmt.Errorf("publish payload too short")
	}

	topicLength := int(binary.BigEndian.Uint16(payload[:2]))
	if len(payload) < 2+topicLength {
		return "", nil, nil, fmt.Errorf("publish topic truncated")
	}

	topic := string(payload[2 : 2+topicLength])
	offset := 2 + topicLength
	qos := (flags >> 1) & 0x03

	var packetID []byte
	if qos > 0 {
		if len(payload) < offset+2 {
			return "", nil, nil, fmt.Errorf("publish packet id truncated")
		}
		packetID = append([]byte(nil), payload[offset:offset+2]...)
		offset += 2
	}

	return topic, packetID, append([]byte(nil), payload[offset:]...), nil
}

func writePacket(writer io.Writer, packetType byte, flags byte, payload []byte) error {
	header := []byte{packetType<<4 | flags}
	header = append(header, encodeRemainingLength(len(payload))...)
	if _, err := writer.Write(header); err != nil {
		return err
	}
	_, err := writer.Write(payload)
	return err
}

func readPacket(reader io.Reader) (byte, byte, []byte, error) {
	var first [1]byte
	if _, err := io.ReadFull(reader, first[:]); err != nil {
		return 0, 0, nil, err
	}

	remainingLength, err := readRemainingLength(reader)
	if err != nil {
		return 0, 0, nil, err
	}

	payload := make([]byte, remainingLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, 0, nil, err
	}

	return first[0] >> 4, first[0] & 0x0F, payload, nil
}

func encodeRemainingLength(length int) []byte {
	var encoded []byte
	for {
		digit := byte(length % 128)
		length /= 128
		if length > 0 {
			digit |= 0x80
		}
		encoded = append(encoded, digit)
		if length == 0 {
			return encoded
		}
	}
}

func readRemainingLength(reader io.Reader) (int, error) {
	multiplier := 1
	value := 0
	for range 4 {
		var digit [1]byte
		if _, err := io.ReadFull(reader, digit[:]); err != nil {
			return 0, err
		}
		value += int(digit[0]&127) * multiplier
		if digit[0]&128 == 0 {
			return value, nil
		}
		multiplier *= 128
	}

	return 0, fmt.Errorf("malformed mqtt remaining length")
}

func writeUTF(buffer *bytes.Buffer, value string) {
	_ = binary.Write(buffer, binary.BigEndian, uint16(len(value)))
	buffer.WriteString(value)
}

func brokerAddress(rawBroker string) string {
	broker := strings.TrimSpace(rawBroker)
	broker = strings.TrimPrefix(broker, "tcp://")
	broker = strings.TrimPrefix(broker, "mqtt://")
	if !strings.Contains(broker, ":") {
		broker += ":1883"
	}
	return broker
}

func (s *Subscriber) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf("[MQTT] "+format, args...)
		return
	}
	log.Printf("[MQTT] "+format, args...)
}
