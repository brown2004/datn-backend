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
