package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (*domain.User, error) {
	const query = `
		INSERT INTO users (phone_number, full_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, phone_number, full_name, password_hash, last_login_at, created_at
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, user.PhoneNumber, user.FullName, user.PasswordHash))
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, email, phone_number, full_name, password_hash, last_login_at, created_at
		FROM users
		WHERE id = $1
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, id))
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, email, phone_number, full_name, password_hash, last_login_at, created_at
		FROM users
		WHERE email = $1
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, email))
}

func (r *UserRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.User, error) {
	const query = `
		SELECT id, email, phone_number, full_name, password_hash, last_login_at, created_at
		FROM users
		WHERE phone_number = $1
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, phoneNumber))
}

func (r *UserRepository) UpdateEmail(ctx context.Context, userID string, email string) (*domain.User, error) {
	const query = `
		UPDATE users
		SET email = $2
		WHERE id = $1
		RETURNING id, email, phone_number, full_name, password_hash, last_login_at, created_at
	`

	return r.scanUser(r.db.QueryRowContext(ctx, query, userID, email))
}

func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	return scanUser(row)
}

func scanUser(row *sql.Row) (*domain.User, error) {
	var user domain.User
	var email sql.NullString
	var lastLoginAt sql.NullTime

	if err := row.Scan(
		&user.ID,
		&email,
		&user.PhoneNumber,
		&user.FullName,
		&user.PasswordHash,
		&lastLoginAt,
		&user.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, err
	}

	if email.Valid {
		user.Email = &email.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}
