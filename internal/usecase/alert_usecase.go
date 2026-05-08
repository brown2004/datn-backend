package usecase

import "datn-backend/internal/domain"

type AlertUseCase struct{}

func NewAlertUseCase() *AlertUseCase {
	return &AlertUseCase{}
}

func (uc *AlertUseCase) CreateAlertFromAgent(agentID string, alertType string, message string) error {
	_ = domain.Alert{
		AgentID: agentID,
		Type:    alertType,
		Message: message,
	}

	return nil
}
