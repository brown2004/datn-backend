package http

import (
	"errors"
	"net/http"

	"datn-backend/internal/domain"
	"datn-backend/internal/usecase"
)

type FeatureHandler struct {
	devices *usecase.DeviceLinkUseCase
}

func NewFeatureHandler(devices *usecase.DeviceLinkUseCase) *FeatureHandler {
	return &FeatureHandler{devices: devices}
}

func (h *FeatureHandler) HandleLinkPCAgent(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req struct {
		DeviceCode string `json:"device_code"`
		DeviceName string `json:"device_name"`
		OSType     string `json:"os_type"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	agent, err := h.devices.LinkPCAgent(r.Context(), domain.LinkPCAgentInput{
		UserID:     userID,
		DeviceCode: req.DeviceCode,
		DeviceName: req.DeviceName,
		OSType:     req.OSType,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrDeviceAlreadyLinked):
			writeError(w, http.StatusConflict, "device_already_linked")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, toPCAgentResponse(agent))
}

func (h *FeatureHandler) HandleListPCAgents(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	agents, err := h.devices.ListPCAgents(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		default:
			writeInternalError(w, err)
		}
		return
	}

	items := make([]any, 0, len(agents))
	for i := range agents {
		items = append(items, toPCAgentResponse(&agents[i]))
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *FeatureHandler) HandleRegisterMobileDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req struct {
		FCMToken string `json:"fcm_token"`
		Platform string `json:"platform"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	device, err := h.devices.RegisterMobileDevice(r.Context(), domain.RegisterMobileDeviceInput{
		UserID:   userID,
		FCMToken: req.FCMToken,
		Platform: req.Platform,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toMobileDeviceResponse(device))
}

func toPCAgentResponse(agent *domain.PCAgent) any {
	var lastSeenAt *string
	if agent.LastSeenAt != nil {
		formatted := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		lastSeenAt = &formatted
	}

	return struct {
		ID               string  `json:"id"`
		DeviceName       string  `json:"device_name"`
		DeviceCode       string  `json:"device_code"`
		OSType           string  `json:"os_type"`
		Status           string  `json:"status"`
		ProtectionStatus string  `json:"protection_status"`
		LastSeenAt       *string `json:"last_seen_at"`
	}{
		ID:               agent.ID,
		DeviceName:       agent.DeviceName,
		DeviceCode:       agent.DeviceCode,
		OSType:           agent.OSType,
		Status:           agent.Status,
		ProtectionStatus: agent.ProtectionStatus,
		LastSeenAt:       lastSeenAt,
	}
}

func toMobileDeviceResponse(device *domain.MobileDevice) any {
	return struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
	}{
		ID:       device.ID,
		Platform: device.Platform,
	}
}
