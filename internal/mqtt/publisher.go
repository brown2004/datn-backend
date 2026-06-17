package mqtt

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

const ProtectionCommandSetProtection = "set_protection"

type ProtectionCommandPayload struct {
	PCAgentID string    `json:"pc_agent_id"`
	Command   string    `json:"command"`
	Enabled   bool      `json:"enabled"`
	Timestamp time.Time `json:"timestamp"`
}

type ProtectionCommandPublisher struct {
	Broker   string
	ClientID string
	Username string
	Password string
}

func ControlTopic(pcAgentID string) string {
	return "pcapp/control/" + strings.TrimSpace(pcAgentID)
}

func (p *ProtectionCommandPublisher) PublishProtectionCommand(ctx context.Context, pcAgentID string, enabled bool) error {
	pcAgentID = strings.TrimSpace(pcAgentID)
	if pcAgentID == "" {
		return fmt.Errorf("pc agent id is empty")
	}

	payload, err := json.Marshal(ProtectionCommandPayload{
		PCAgentID: pcAgentID,
		Command:   ProtectionCommandSetProtection,
		Enabled:   enabled,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal protection command payload: %w", err)
	}

	return p.Publish(ctx, ControlTopic(pcAgentID), payload)
}

func (p *ProtectionCommandPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	if strings.TrimSpace(p.Broker) == "" {
		return fmt.Errorf("mqtt broker is empty")
	}
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("mqtt topic is empty")
	}

	addr := brokerAddress(p.Broker)
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

	subscriber := &Subscriber{
		ClientID: publisherClientID(p.ClientID),
		Username: p.Username,
		Password: p.Password,
	}
	if err := subscriber.connect(conn); err != nil {
		return err
	}

	if err := publishQoS1(conn, topic, payload); err != nil {
		return err
	}

	_ = writePacket(conn, 14, 0, nil)
	return nil
}

func publishQoS1(conn net.Conn, topic string, payload []byte) error {
	var packet bytes.Buffer
	writeUTF(&packet, topic)
	_ = binary.Write(&packet, binary.BigEndian, uint16(1))
	packet.Write(payload)

	if err := writePacket(conn, 3, 2, packet.Bytes()); err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	packetType, _, response, err := readPacket(conn)
	if err != nil {
		return err
	}
	if packetType != 4 || len(response) != 2 || binary.BigEndian.Uint16(response) != 1 {
		return fmt.Errorf("invalid mqtt puback")
	}
	_ = conn.SetReadDeadline(time.Time{})

	return nil
}

func publisherClientID(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "datn-backend"
	}

	return fmt.Sprintf("%s-control-%d", base, time.Now().UnixNano())
}
