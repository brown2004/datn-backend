package postgres

import (
	"context"

	"datn-backend/internal/domain"
)

type MobileDeviceRepository struct{}

func NewMobileDeviceRepository() *MobileDeviceRepository {
	return &MobileDeviceRepository{}
}

func (r *MobileDeviceRepository) FindByUserID(ctx context.Context, userID string) ([]domain.MobileDevice, error) {
	return nil, nil
}

func (r *MobileDeviceRepository) Save(ctx context.Context, device *domain.MobileDevice) error {
	return nil
}
