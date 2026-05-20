package repo

import (
	"context"

	"datn-backend/internal/domain"
)

type RegistrationRepository interface {
	CompleteRegister(ctx context.Context, user domain.User, refreshToken domain.RefreshToken) (*domain.User, error)
}
