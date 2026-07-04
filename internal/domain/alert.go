package domain

import "time"

const (
	AlertTypeUSBRemoved          = "usb_removed"
	AlertTypeMotionDetected      = "motion_detected"
	AlertTypePCAgentDisconnected = "pc_agent_disconnected"
)

type Alert struct {
	ID        string
	AgentID   string
	AgentName string
	UserID    string
	Type      string
	Message   string
	CreatedAt time.Time
}

func AlertTitle(alertType string) string {
	switch alertType {
	case AlertTypeMotionDetected:
		return "Phát hiện rung lắc"
	case AlertTypePCAgentDisconnected:
		return "Mất kết nối đột ngột"
	case AlertTypeUSBRemoved:
		return "Thiết bị bị ngắt kết nối"
	default:
		return "Cảnh báo mới"
	}
}

func IsKnownAlertType(alertType string) bool {
	switch alertType {
	case AlertTypeMotionDetected,
		AlertTypePCAgentDisconnected,
		AlertTypeUSBRemoved:
		return true
	default:
		return false
	}
}

func AlertMessage(alertType string) string {
	switch alertType {
	case AlertTypeMotionDetected:
		return "Phát hiện rung lắc hoặc di chuyển bất thường."
	case AlertTypePCAgentDisconnected:
		return "PC Agent đã mất kết nối đột ngột với hệ thống."
	case AlertTypeUSBRemoved:
		return "Thiết bị bảo vệ đã bị ngắt kết nối."
	default:
		return "Phát hiện cảnh báo mới từ thiết bị."
	}
}
