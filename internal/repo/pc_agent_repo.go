package repo

import (
	"context"

	"datn-backend/internal/domain"
)

type PCAgentRepository interface {
	FindByID(ctx context.Context, id string) (*domain.PCAgent, error)
	Save(ctx context.Context, agent *domain.PCAgent) error
}
