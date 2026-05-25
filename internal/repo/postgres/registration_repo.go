package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"

	"github.com/jackc/pgx/v5/pgconn"
)

type RegistrationRepository struct {
	db *sql.DB
}

func NewRegistrationRepository(db *sql.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) CompleteRegister(ctx context.Context, user domain.User, refreshToken domain.RefreshToken) (*domain.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	createdUser, err := scanUser(tx.QueryRowContext(
		ctx,
		`
			INSERT INTO users (phone_number, full_name, password_hash)
			VALUES ($1, $2, $3)
			RETURNING id, email, phone_number, full_name, password_hash, last_login_at, created_at
		`,
		user.PhoneNumber,
		user.FullName,
		user.PasswordHash,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, repo.ErrUserAlreadyExists
		}
		return nil, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)
		`,
		createdUser.ID,
		refreshToken.TokenHash,
		refreshToken.ExpiresAt,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return createdUser, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
