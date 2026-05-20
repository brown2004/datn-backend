package domain

type LinkPCAgentInput struct {
	UserID     string
	DeviceCode string
	DeviceName string
	OSType     string
}

type RegisterMobileDeviceInput struct {
	UserID   string
	FCMToken string
	Platform string
}
