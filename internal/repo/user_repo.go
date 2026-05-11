package repo

import (
	"context"
	"errors"

	"datn-backend/internal/domain"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user domain.User) (*domain.User, error)
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	UpdateEmail(ctx context.Context, userID string, email string) (*domain.User, error)
}
