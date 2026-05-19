package domain

import "time"

type MobileDevice struct {
	ID        string
	UserID    string
	FCMToken  string
	Platform  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
