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

func (s *MockSender) SendAlert(ctx context.Context, alert domain.Alert) error {
	log.Printf("mock send alert: agent_id=%s type=%s message=%s", alert.AgentID, alert.Type, alert.Message)
	return nil
}
