package repo

import (
	"context"
	"errors"
	"time"

	"datn-backend/internal/domain"
)

var ErrPCAgentNotFound = errors.New("pc agent not found")

type PCAgentRepository interface {
	Create(ctx context.Context, agent domain.PCAgent) (*domain.PCAgent, error)
	FindByID(ctx context.Context, id string) (*domain.PCAgent, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.PCAgent, error)
	SetAgentSecretHashIfEmpty(ctx context.Context, id string, secretHash string) (*domain.PCAgent, bool, error)
	MarkVerified(ctx context.Context, id string, verifiedAt time.Time) (*domain.PCAgent, error)
	UpdateProtectionStatusByIDAndUserID(ctx context.Context, id string, userID string, protectionStatus string) (*domain.PCAgent, error)
	DeleteByIDAndUserID(ctx context.Context, id string, userID string) error
	Save(ctx context.Context, agent *domain.PCAgent) error
}
