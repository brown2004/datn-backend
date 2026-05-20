package repo

import (
	"context"
	"errors"

	"datn-backend/internal/domain"
)

var ErrPCAgentNotFound = errors.New("pc agent not found")
var ErrPCAgentAlreadyLinked = errors.New("pc agent already linked")

type PCAgentRepository interface {
	Create(ctx context.Context, agent domain.PCAgent) (*domain.PCAgent, error)
	FindByID(ctx context.Context, id string) (*domain.PCAgent, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.PCAgent, error)
	Save(ctx context.Context, agent *domain.PCAgent) error
}
