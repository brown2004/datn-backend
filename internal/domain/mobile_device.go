package domain

import "time"

type MobileDevice struct {
	ID        string
	UserID    string
	Name      string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
