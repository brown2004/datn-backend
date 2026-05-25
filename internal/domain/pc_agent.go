package domain

import "time"

const (
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"

	ProtectionStatusEnabled  = "enabled"
	ProtectionStatusDisabled = "disabled"
)

type PCAgent struct {
	ID               string
	UserID           string
	DeviceName       string
	OSType           string
	AgentSecretHash  *string
	Status           string
	ProtectionStatus string
	LastSeenAt       *time.Time
	CreatedAt        time.Time
}
