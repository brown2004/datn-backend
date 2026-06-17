package notification

import (
	"context"
	"log"

	"datn-backend/internal/domain"
)

type MockSender struct{}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (s *MockSender) SendAlert(ctx context.Context, alert domain.Alert, devices []domain.MobileDevice) (SendReport, error) {
	log.Printf("mock send alert: agent_id=%s type=%s message=%s target_devices=%d", alert.AgentID, alert.Type, alert.Message, len(devices))
	return SendReport{Targeted: len(devices), Sent: len(devices)}, nil
}
