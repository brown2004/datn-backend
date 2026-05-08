package postgres

import (
	"context"

	"datn-backend/internal/domain"
)

type PCAgentRepository struct{}

func NewPCAgentRepository() *PCAgentRepository {
	return &PCAgentRepository{}
}

func (r *PCAgentRepository) FindByID(ctx context.Context, id string) (*domain.PCAgent, error) {
	return nil, nil
}

func (r *PCAgentRepository) Save(ctx context.Context, agent *domain.PCAgent) error {
	return nil
}
