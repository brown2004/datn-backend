package postgres

import (
	"context"

	"datn-backend/internal/domain"
)

type AlertRepository struct{}

func NewAlertRepository() *AlertRepository {
	return &AlertRepository{}
}

func (r *AlertRepository) Save(ctx context.Context, alert *domain.Alert) error {
	return nil
}
