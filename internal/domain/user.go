package domain

import "time"

type User struct {
	ID           string
	Email        *string
	PhoneNumber  string
	FullName     string
	PasswordHash string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}
