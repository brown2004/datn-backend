package repo

import (
	"context"
	"time"

	"datn-backend/internal/domain"
)

type AlertRepository interface {
	Save(ctx context.Context, alert *domain.Alert) error
	FindByUserID(ctx context.Context, userID string) ([]domain.Alert, error)
	FindRecentByAgentAndType(ctx context.Context, agentID string, alertType string, since time.Time) (*domain.Alert, error)
}
