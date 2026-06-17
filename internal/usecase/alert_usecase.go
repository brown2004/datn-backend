package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"datn-backend/internal/domain"
	"datn-backend/internal/notification"
	"datn-backend/internal/repo"
)

const duplicateAlertWindow = 5 * time.Second

type AlertUseCase struct {
	alerts        repo.AlertRepository
	pcAgents      repo.PCAgentRepository
	mobileDevices repo.MobileDeviceRepository
	notifications notification.Sender
}

func NewAlertUseCase(
	alerts repo.AlertRepository,
	pcAgents repo.PCAgentRepository,
	mobileDevices repo.MobileDeviceRepository,
	notifications notification.Sender,
) *AlertUseCase {
	return &AlertUseCase{
		alerts:        alerts,
		pcAgents:      pcAgents,
		mobileDevices: mobileDevices,
		notifications: notifications,
	}
}

type CreateAlertFromAgentInput struct {
	PCAgentID   string
	AlertType   string
	Message     string
	TriggeredAt time.Time
}

func (uc *AlertUseCase) CreateAlertFromAgent(ctx context.Context, input CreateAlertFromAgentInput) (*domain.Alert, error) {
	pcAgentID := strings.TrimSpace(input.PCAgentID)
	alertType := normalizeAlertType(input.AlertType)
	if pcAgentID == "" || alertType == "" {
		return nil, ErrInvalidInput
	}

	agent, err := uc.pcAgents.FindByID(ctx, pcAgentID)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, ErrPCAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(agent.UserID) == "" {
		return nil, ErrPCAgentNotFound
	}

	triggeredAt := input.TriggeredAt
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}
	triggeredAt = triggeredAt.UTC()

	recentAlert, err := uc.alerts.FindRecentByAgentAndType(ctx, agent.ID, alertType, triggeredAt.Add(-duplicateAlertWindow))
	if err != nil {
		return nil, err
	}
	if recentAlert != nil {
		log.Printf("duplicate alert skipped: existing_alert_id=%s pc_agent_id=%s type=%s window=%s", recentAlert.ID, agent.ID, alertType, duplicateAlertWindow)
		return recentAlert, nil
	}

	alert := &domain.Alert{
		AgentID:   agent.ID,
		UserID:    agent.UserID,
		Type:      alertType,
		Message:   alertMessage(alertType, input.Message),
		CreatedAt: triggeredAt,
	}
	if err := uc.alerts.Save(ctx, alert); err != nil {
		return nil, err
	}

	devices, err := uc.mobileDevices.FindByUserID(ctx, agent.UserID)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 { // khong co user nao lien ket voi thiet bi nay
		log.Printf("alert notification skipped: no mobile devices user_id=%s alert_id=%s", agent.UserID, alert.ID)
		return alert, nil
	}

	if len(devices) > 0 && uc.notifications != nil {
		report, err := uc.notifications.SendAlert(ctx, *alert, devices)
		invalidTokensRemoved, nonInvalidFailures := uc.cleanupInvalidNotificationTokens(ctx, agent.UserID, report.Failed)
		if nonInvalidErr := notificationFailuresError(nonInvalidFailures); nonInvalidErr != nil {
			log.Printf("alert notification failed: alert_id=%s user_id=%s devices=%d sent=%d invalid_tokens_removed=%d error=%v", alert.ID, agent.UserID, len(devices), report.Sent, invalidTokensRemoved, nonInvalidErr)
			return alert, nonInvalidErr
		}
		if err != nil && len(report.Failed) == 0 {
			log.Printf("alert notification failed: alert_id=%s user_id=%s devices=%d error=%v", alert.ID, agent.UserID, len(devices), err)
			return alert, err
		}
		log.Printf("alert notification sent: alert_id=%s user_id=%s devices=%d targeted=%d sent=%d invalid_tokens_removed=%d", alert.ID, agent.UserID, len(devices), report.Targeted, report.Sent, invalidTokensRemoved)
	}

	return alert, nil
}

func (uc *AlertUseCase) cleanupInvalidNotificationTokens(ctx context.Context, userID string, failures []notification.SendFailure) (int, []notification.SendFailure) {
	var nonInvalidFailures []notification.SendFailure
	removedTokens := make(map[string]struct{})

	for _, failure := range failures {
		if !failure.InvalidToken {
			nonInvalidFailures = append(nonInvalidFailures, failure)
			continue
		}

		token := strings.TrimSpace(failure.Token)
		if token == "" {
			continue
		}
		if _, ok := removedTokens[token]; ok {
			continue
		}

		if err := uc.mobileDevices.DeleteByFCMToken(ctx, token); err != nil {
			log.Printf("invalid fcm token cleanup failed: user_id=%s token=%s error=%v", userID, tokenPrefix(token), err)
			continue
		}

		removedTokens[token] = struct{}{}
		log.Printf("invalid fcm token removed: user_id=%s token=%s", userID, tokenPrefix(token))
	}

	return len(removedTokens), nonInvalidFailures
}

func notificationFailuresError(failures []notification.SendFailure) error {
	var errs []error
	for _, failure := range failures {
		if failure.Err != nil {
			errs = append(errs, failure.Err)
		}
	}
	return errors.Join(errs...)
}

func tokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 12 {
		return token
	}
	return token[:12] + "..."
}

func (uc *AlertUseCase) ListAlerts(ctx context.Context, userID string) ([]domain.Alert, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}

	return uc.alerts.FindByUserID(ctx, userID)
}

func normalizeAlertType(alertType string) string {
	switch strings.TrimSpace(strings.ToLower(alertType)) {
	case "motion", "motion_alert", "motion_detected":
		return domain.AlertTypeMotionDetected
	case "usb_removed", "usb_disconnected":
		return domain.AlertTypeUSBRemoved
	default:
		return strings.TrimSpace(strings.ToLower(alertType))
	}
}

func alertMessage(alertType string, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}

	switch alertType {
	case domain.AlertTypeMotionDetected:
		return "Phát hiện rung lắc hoặc di chuyển bất thường."
	case domain.AlertTypeUSBRemoved:
		return "Thiết bị bảo vệ đã bị ngắt kết nối."
	default:
		return "Phát hiện cảnh báo mới từ thiết bị."
	}
}
