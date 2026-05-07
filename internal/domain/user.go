package domain

import "time"

type User struct {
	ID        string
	Email     string
	FullName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
