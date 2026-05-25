package repo

import (
	"context"
	"errors"

	"datn-backend/internal/domain"
)

var ErrOTPNotFound = errors.New("otp not found")

type AuthOTPRepository interface {
	Create(ctx context.Context, otp domain.AuthOTP) (*domain.AuthOTP, error)
	FindLatest(ctx context.Context, phoneNumber string, purpose string) (*domain.AuthOTP, error)
	IncrementAttempt(ctx context.Context, id string) error
	MarkUsed(ctx context.Context, id string) error
}
