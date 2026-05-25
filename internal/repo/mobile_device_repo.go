package repo

import (
	"context"

	"datn-backend/internal/domain"
)

type MobileDeviceRepository interface {
	FindByUserID(ctx context.Context, userID string) ([]domain.MobileDevice, error)
	Upsert(ctx context.Context, device domain.MobileDevice) (*domain.MobileDevice, error)
	Save(ctx context.Context, device *domain.MobileDevice) error
}
