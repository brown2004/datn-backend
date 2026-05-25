package repo

import (
	"context"
	"errors"
	"time"

	"datn-backend/internal/domain"
)

var ErrPairingSessionNotFound = errors.New("pairing session not found")
var ErrPairingSessionNotPending = errors.New("pairing session not pending")
var ErrPairingSessionExpired = errors.New("pairing session expired")
var ErrPairingSessionCodeExists = errors.New("pairing session code exists")

type PairingSessionRepository interface {
	Create(ctx context.Context, session domain.PairingSession) (*domain.PairingSession, error)
	FindByDeviceCode(ctx context.Context, deviceCode string) (*domain.PairingSession, error)
	FindByIDAndDeviceCode(ctx context.Context, id string, deviceCode string) (*domain.PairingSession, error)
	ExistsByDeviceCode(ctx context.Context, deviceCode string) (bool, error)
	Expire(ctx context.Context, id string) (*domain.PairingSession, error)
	Confirm(ctx context.Context, deviceCode string, userID string, now time.Time) (*domain.PCAgent, *domain.PairingSession, error)
}
