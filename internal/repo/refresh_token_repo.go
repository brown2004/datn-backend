package repo

import (
	"context"
	"errors"

	"datn-backend/internal/domain"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken domain.RefreshToken) (*domain.RefreshToken, error)
	FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	Replace(ctx context.Context, currentTokenHash string, nextToken domain.RefreshToken) (*domain.RefreshToken, error)
	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}
