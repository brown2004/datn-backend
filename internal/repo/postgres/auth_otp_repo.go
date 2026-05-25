package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

type AuthOTPRepository struct {
	db *sql.DB
}

func NewAuthOTPRepository(db *sql.DB) *AuthOTPRepository {
	return &AuthOTPRepository{db: db}
}

func (r *AuthOTPRepository) Create(ctx context.Context, otp domain.AuthOTP) (*domain.AuthOTP, error) {
	const query = `
		INSERT INTO auth_otps (phone_number, purpose, otp_hash, expires_at, attempt_count, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, phone_number, purpose, otp_hash, expires_at, attempt_count, max_attempts, used_at, created_at
	`

	return r.scanOTP(r.db.QueryRowContext(
		ctx,
		query,
		otp.PhoneNumber,
		otp.Purpose,
		otp.OTPHash,
		otp.ExpiresAt,
		otp.AttemptCount,
		otp.MaxAttempts,
	))
}

func (r *AuthOTPRepository) FindLatest(ctx context.Context, phoneNumber string, purpose string) (*domain.AuthOTP, error) {
	const query = `
		SELECT id, phone_number, purpose, otp_hash, expires_at, attempt_count, max_attempts, used_at, created_at
		FROM auth_otps
		WHERE phone_number = $1 AND purpose = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanOTP(r.db.QueryRowContext(ctx, query, phoneNumber, purpose))
}

func (r *AuthOTPRepository) IncrementAttempt(ctx context.Context, id string) error {
	const query = `
		UPDATE auth_otps
		SET attempt_count = attempt_count + 1
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repo.ErrOTPNotFound
	}

	return nil
}

func (r *AuthOTPRepository) MarkUsed(ctx context.Context, id string) error {
	const query = `
		UPDATE auth_otps
		SET used_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repo.ErrOTPNotFound
	}

	return nil
}

func (r *AuthOTPRepository) scanOTP(row *sql.Row) (*domain.AuthOTP, error) {
	var otp domain.AuthOTP
	var usedAt sql.NullTime

	if err := row.Scan(
		&otp.ID,
		&otp.PhoneNumber,
		&otp.Purpose,
		&otp.OTPHash,
		&otp.ExpiresAt,
		&otp.AttemptCount,
		&otp.MaxAttempts,
		&usedAt,
		&otp.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrOTPNotFound
		}
		return nil, err
	}

	if usedAt.Valid {
		otp.UsedAt = &usedAt.Time
	}

	return &otp, nil
}
