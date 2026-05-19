package usecase

import (
	"context"
	"errors"
	"strings"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

var (
	ErrDeviceAlreadyLinked = errors.New("device already linked")
)

type DeviceLinkUseCase struct {
	pcAgents      repo.PCAgentRepository
	mobileDevices repo.MobileDeviceRepository
}

func NewDeviceLinkUseCase(pcAgents repo.PCAgentRepository, mobileDevices repo.MobileDeviceRepository) *DeviceLinkUseCase {
	return &DeviceLinkUseCase{
		pcAgents:      pcAgents,
		mobileDevices: mobileDevices,
	}
}

func (uc *DeviceLinkUseCase) LinkPCAgent(ctx context.Context, input domain.LinkPCAgentInput) (*domain.PCAgent, error) {
	userID := strings.TrimSpace(input.UserID)
	deviceCode := strings.TrimSpace(input.DeviceCode)
	deviceName := strings.TrimSpace(input.DeviceName)
	osType := strings.TrimSpace(strings.ToLower(input.OSType))

	if userID == "" || deviceCode == "" || deviceName == "" || osType == "" {
		return nil, ErrInvalidInput
	}

	agent, err := uc.pcAgents.Create(ctx, domain.PCAgent{
		UserID:           userID,
		DeviceCode:       deviceCode,
		DeviceName:       deviceName,
		OSType:           osType,
		Status:           domain.AgentStatusOffline,
		ProtectionStatus: domain.ProtectionStatusDisabled,
	})
	if errors.Is(err, repo.ErrPCAgentAlreadyLinked) {
		return nil, ErrDeviceAlreadyLinked
	}
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (uc *DeviceLinkUseCase) ListPCAgents(ctx context.Context, userID string) ([]domain.PCAgent, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}

	return uc.pcAgents.FindByUserID(ctx, userID)
}

func (uc *DeviceLinkUseCase) RegisterMobileDevice(ctx context.Context, input domain.RegisterMobileDeviceInput) (*domain.MobileDevice, error) {
	userID := strings.TrimSpace(input.UserID)
	fcmToken := strings.TrimSpace(input.FCMToken)
	platform := strings.TrimSpace(strings.ToLower(input.Platform))

	if userID == "" || fcmToken == "" || platform == "" {
		return nil, ErrInvalidInput
	}

	return uc.mobileDevices.Upsert(ctx, domain.MobileDevice{
		UserID:   userID,
		FCMToken: fcmToken,
		Platform: platform,
	})
}
