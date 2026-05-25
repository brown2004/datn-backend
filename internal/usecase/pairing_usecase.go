package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"
	"time"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPairingSessionNotFound   = errors.New("pairing session not found")
	ErrPairingSessionMismatch   = errors.New("pairing session mismatch")
	ErrPairingSessionNotPending = errors.New("pairing session not pending")
	ErrPairingSessionExpired    = errors.New("pairing session expired")
	ErrPairingCodeUnavailable   = errors.New("pairing code unavailable")
	ErrAgentCredentialInvalid   = errors.New("agent credential invalid")
	ErrPCAgentNotFound          = errors.New("pc agent not found")
)

const (
	pairingSessionTTL    = 10 * time.Minute
	deviceCodeLength     = 6
	deviceCodeMaxRetries = 10
	deviceCodeAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	agentSecretBytes     = 32
)

type PairingUseCase struct {
	pcAgents        repo.PCAgentRepository
	mobileDevices   repo.MobileDeviceRepository
	pairingSessions repo.PairingSessionRepository
}

func NewPairingUseCase(pcAgents repo.PCAgentRepository, mobileDevices repo.MobileDeviceRepository, pairingSessions repo.PairingSessionRepository) *PairingUseCase {
	return &PairingUseCase{
		pcAgents:        pcAgents,
		mobileDevices:   mobileDevices,
		pairingSessions: pairingSessions,
	}
}

type PairingStatusResult struct {
	Status           string
	PCAgentID        *string
	AgentSecret      string
	CredentialIssued bool
}

func (uc *PairingUseCase) StartPairing(ctx context.Context, input domain.StartPCAgentPairingInput) (*domain.PairingSession, error) {
	deviceName := strings.TrimSpace(input.DeviceName)
	osType := strings.TrimSpace(strings.ToLower(input.OSType))

	if deviceName == "" || osType == "" {
		return nil, ErrInvalidInput
	}

	for range deviceCodeMaxRetries {
		deviceCode, err := generateDeviceCode()
		if err != nil {
			return nil, err
		}

		exists, err := uc.pairingSessions.ExistsByDeviceCode(ctx, deviceCode)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		session, err := uc.pairingSessions.Create(ctx, domain.PairingSession{
			DeviceCode: deviceCode,
			DeviceName: deviceName,
			OSType:     osType,
			Status:     domain.PairingStatusPending,
			ExpiresAt:  time.Now().UTC().Add(pairingSessionTTL),
		})
		if errors.Is(err, repo.ErrPairingSessionCodeExists) {
			continue
		}

		return session, err
	}

	return nil, ErrPairingCodeUnavailable
}

func (uc *PairingUseCase) ConfirmPairing(ctx context.Context, input domain.ConfirmPCAgentPairingInput) (*domain.PCAgent, error) {
	userID := strings.TrimSpace(input.UserID)
	deviceCode := strings.TrimSpace(strings.ToUpper(input.DeviceCode))
	if userID == "" || deviceCode == "" {
		return nil, ErrInvalidInput
	}

	agent, _, err := uc.pairingSessions.Confirm(ctx, deviceCode, userID, time.Now().UTC())
	if errors.Is(err, repo.ErrPairingSessionNotFound) {
		return nil, ErrPairingSessionNotFound
	}
	if errors.Is(err, repo.ErrPairingSessionNotPending) {
		return nil, ErrPairingSessionNotPending
	}
	if errors.Is(err, repo.ErrPairingSessionExpired) {
		return nil, ErrPairingSessionExpired
	}
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (uc *PairingUseCase) GetPairingStatus(ctx context.Context, input domain.PairingStatusInput) (*PairingStatusResult, error) {
	pairingSessionID := strings.TrimSpace(input.PairingSessionID)
	deviceCode := strings.TrimSpace(strings.ToUpper(input.DeviceCode))
	if pairingSessionID == "" || deviceCode == "" {
		return nil, ErrInvalidInput
	}

	session, err := uc.pairingSessions.FindByIDAndDeviceCode(ctx, pairingSessionID, deviceCode)
	if errors.Is(err, repo.ErrPairingSessionNotFound) {
		exists, existsErr := uc.pairingSessions.ExistsByDeviceCode(ctx, deviceCode)
		if existsErr != nil {
			return nil, existsErr
		}
		if !exists {
			return nil, ErrPairingSessionNotFound
		}
		return nil, ErrPairingSessionMismatch
	}
	if err != nil {
		return nil, err
	}

	if session.Status == domain.PairingStatusPending && time.Now().UTC().After(session.ExpiresAt) {
		session, err = uc.pairingSessions.Expire(ctx, session.ID)
		if errors.Is(err, repo.ErrPairingSessionNotFound) {
			current, findErr := uc.pairingSessions.FindByIDAndDeviceCode(ctx, pairingSessionID, deviceCode)
			if errors.Is(findErr, repo.ErrPairingSessionNotFound) {
				return nil, ErrPairingSessionNotFound
			}
			if findErr != nil {
				return nil, findErr
			}
			session = current
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}

	result := &PairingStatusResult{Status: session.Status}
	if session.Status != domain.PairingStatusConfirmed || session.PCAgentID == nil {
		return result, nil
	}

	result.PCAgentID = session.PCAgentID

	agent, err := uc.pcAgents.FindByID(ctx, *session.PCAgentID)
	if err != nil {
		return nil, err
	}
	if agent.AgentSecretHash != nil {
		result.CredentialIssued = true
		return result, nil
	}

	agentSecret, err := generateAgentSecret()
	if err != nil {
		return nil, err
	}
	agentSecretHash, err := hashAgentSecret(agentSecret)
	if err != nil {
		return nil, err
	}

	_, issued, err := uc.pcAgents.SetAgentSecretHashIfEmpty(ctx, agent.ID, agentSecretHash)
	if err != nil {
		return nil, err
	}
	if !issued {
		result.CredentialIssued = true
		return result, nil
	}

	result.AgentSecret = agentSecret
	return result, nil
}

func (uc *PairingUseCase) VerifyPCAgent(ctx context.Context, input domain.VerifyPCAgentInput) (*domain.PCAgent, error) {
	pcAgentID := strings.TrimSpace(input.PCAgentID)
	agentSecret := strings.TrimSpace(input.AgentSecret)
	if pcAgentID == "" || agentSecret == "" {
		return nil, ErrAgentCredentialInvalid
	}

	agent, err := uc.pcAgents.FindByID(ctx, pcAgentID)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, ErrAgentCredentialInvalid
	}
	if err != nil {
		return nil, err
	}
	if agent.AgentSecretHash == nil {
		return nil, ErrAgentCredentialInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*agent.AgentSecretHash), []byte(agentSecret)); err != nil {
		return nil, ErrAgentCredentialInvalid
	}

	return uc.pcAgents.MarkVerified(ctx, agent.ID, time.Now().UTC())
}

func (uc *PairingUseCase) UpdatePCAgentOwnProtection(ctx context.Context, input domain.UpdatePCAgentOwnProtectionInput) (*domain.PCAgent, error) {
	pcAgentID := strings.TrimSpace(input.PCAgentID)
	agentSecret := strings.TrimSpace(input.AgentSecret)
	if pcAgentID == "" || agentSecret == "" {
		return nil, ErrAgentCredentialInvalid
	}

	agent, err := uc.pcAgents.FindByID(ctx, pcAgentID)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, ErrAgentCredentialInvalid
	}
	if err != nil {
		return nil, err
	}
	if agent.AgentSecretHash == nil {
		return nil, ErrAgentCredentialInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*agent.AgentSecretHash), []byte(agentSecret)); err != nil {
		return nil, ErrAgentCredentialInvalid
	}

	protectionStatus := domain.ProtectionStatusDisabled
	if input.Enabled {
		protectionStatus = domain.ProtectionStatusEnabled
	}

	agent, err = uc.pcAgents.UpdateProtectionStatusByID(ctx, agent.ID, protectionStatus)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, ErrAgentCredentialInvalid
	}
	if err != nil {
		return nil, err
	}

	return uc.pcAgents.MarkVerified(ctx, agent.ID, time.Now().UTC())
}

func (uc *PairingUseCase) ListPCAgents(ctx context.Context, userID string) ([]domain.PCAgent, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}

	return uc.pcAgents.FindByUserID(ctx, userID)
}

func (uc *PairingUseCase) DeletePCAgent(ctx context.Context, input domain.DeletePCAgentInput) error {
	userID := strings.TrimSpace(input.UserID)
	pcAgentID := strings.TrimSpace(input.PCAgentID)
	if userID == "" || pcAgentID == "" {
		return ErrInvalidInput
	}

	err := uc.pcAgents.DeleteByIDAndUserID(ctx, pcAgentID, userID)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return ErrPCAgentNotFound
	}

	return err
}

func (uc *PairingUseCase) UpdatePCAgentProtection(ctx context.Context, input domain.UpdatePCAgentProtectionInput) (*domain.PCAgent, error) {
	userID := strings.TrimSpace(input.UserID)
	pcAgentID := strings.TrimSpace(input.PCAgentID)
	if userID == "" || pcAgentID == "" {
		return nil, ErrInvalidInput
	}

	protectionStatus := domain.ProtectionStatusDisabled
	if input.Enabled {
		protectionStatus = domain.ProtectionStatusEnabled
	}

	agent, err := uc.pcAgents.UpdateProtectionStatusByIDAndUserID(ctx, pcAgentID, userID, protectionStatus)
	if errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, ErrPCAgentNotFound
	}
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (uc *PairingUseCase) RegisterMobileDevice(ctx context.Context, input domain.RegisterMobileDeviceInput) (*domain.MobileDevice, error) {
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

func generateDeviceCode() (string, error) {
	max := big.NewInt(int64(len(deviceCodeAlphabet)))
	code := make([]byte, deviceCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = deviceCodeAlphabet[n.Int64()]
	}

	return string(code), nil
}

func generateAgentSecret() (string, error) {
	bytes := make([]byte, agentSecretBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashAgentSecret(agentSecret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(agentSecret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
