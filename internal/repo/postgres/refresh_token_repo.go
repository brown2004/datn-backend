package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, refreshToken domain.RefreshToken) (*domain.RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at
	`

	return r.scanRefreshToken(r.db.QueryRowContext(
		ctx,
		query,
		refreshToken.UserID,
		refreshToken.TokenHash,
		refreshToken.ExpiresAt,
	))
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	return r.scanRefreshToken(r.db.QueryRowContext(ctx, query, tokenHash))
}

func (r *RefreshTokenRepository) Replace(ctx context.Context, currentTokenHash string, nextToken domain.RefreshToken) (*domain.RefreshToken, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(
		ctx,
		`
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE token_hash = $1 AND revoked_at IS NULL
		`,
		currentTokenHash,
	)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, repo.ErrRefreshTokenNotFound
	}

	next, err := r.scanRefreshToken(tx.QueryRowContext(
		ctx,
		`
			INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)
			RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at
		`,
		nextToken.UserID,
		nextToken.TokenHash,
		nextToken.ExpiresAt,
	))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return next, nil
}

func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repo.ErrRefreshTokenNotFound
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

type refreshTokenRow interface {
	Scan(dest ...any) error
}

func (r *RefreshTokenRepository) scanRefreshToken(row refreshTokenRow) (*domain.RefreshToken, error) {
	var refreshToken domain.RefreshToken
	var revokedAt sql.NullTime

	if err := row.Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.TokenHash,
		&refreshToken.ExpiresAt,
		&revokedAt,
		&refreshToken.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	if revokedAt.Valid {
		refreshToken.RevokedAt = &revokedAt.Time
	}

	return &refreshToken, nil
}
