package repo

import (
	"context"

	"datn-backend/internal/domain"
)

type AlertRepository interface {
	Save(ctx context.Context, alert *domain.Alert) error
}
