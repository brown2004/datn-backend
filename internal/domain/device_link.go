package domain

type StartPCAgentPairingInput struct {
	PCAgentID  string
	DeviceName string
	OSType     string
}

type ConfirmPCAgentPairingInput struct {
	UserID     string
	DeviceCode string
}

type PairingStatusInput struct {
	PairingSessionID string
	DeviceCode       string
}

type VerifyPCAgentInput struct {
	PCAgentID   string
	AgentSecret string
}

type DeletePCAgentInput struct {
	UserID    string
	PCAgentID string
}

type UpdatePCAgentProtectionInput struct {
	UserID    string
	PCAgentID string
	Enabled   bool
}

type UpdatePCAgentOwnProtectionInput struct {
	PCAgentID   string
	AgentSecret string
	Enabled     bool
}

type RegisterMobileDeviceInput struct {
	UserID   string
	FCMToken string
	Platform string
}
