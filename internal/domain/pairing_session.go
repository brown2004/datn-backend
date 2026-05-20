package domain

import "time"

const (
	PairingStatusPending   = "pending"
	PairingStatusConfirmed = "confirmed"
	PairingStatusExpired   = "expired"
	PairingStatusCancelled = "cancelled"
)

type PairingSession struct {
	ID                string
	DeviceCode        string
	DeviceName        string
	OSType            string
	Status            string
	ExpiresAt         time.Time
	ConfirmedByUserID *string
	PCAgentID         *string
	ConfirmedAt       *time.Time
	CreatedAt         time.Time
}
