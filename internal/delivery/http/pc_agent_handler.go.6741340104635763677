package http

import (
	"errors"
	"net/http"
	"time"

	"datn-backend/internal/domain"
	"datn-backend/internal/usecase"
)

type PCAgentHandler struct {
	pairing *usecase.PairingUseCase
}

func NewPCAgentHandler(pairing *usecase.PairingUseCase) *PCAgentHandler {
	return &PCAgentHandler{pairing: pairing}
}

func (h *PCAgentHandler) HandleStartPairing(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceName string `json:"device_name"`
		OSType     string `json:"os_type"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	session, err := h.pairing.StartPairing(r.Context(), domain.StartPCAgentPairingInput{
		DeviceName: req.DeviceName,
		OSType:     req.OSType,
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

	writeJSON(w, http.StatusCreated, struct {
		PairingSessionID string `json:"pairing_session_id"`
		DeviceCode       string `json:"device_code"`
		ExpiresIn        int64  `json:"expires_in"`
	}{
		PairingSessionID: session.ID,
		DeviceCode:       session.DeviceCode,
		ExpiresIn:        secondsUntil(session.ExpiresAt),
	})
}

func (h *PCAgentHandler) HandleConfirmPairing(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	agent, err := h.pairing.ConfirmPairing(r.Context(), domain.ConfirmPCAgentPairingInput{
		UserID:     userID,
		DeviceCode: req.DeviceCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPairingSessionNotFound):
			writeMessageError(w, http.StatusBadRequest, "Mã liên kết không hợp lệ")
		case errors.Is(err, usecase.ErrPairingSessionNotPending):
			writeMessageError(w, http.StatusBadRequest, "Mã liên kết đã được sử dụng hoặc không còn hiệu lực")
		case errors.Is(err, usecase.ErrPairingSessionExpired):
			writeMessageError(w, http.StatusBadRequest, "Mã liên kết đã hết hạn")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, struct {
		Message string `json:"message"`
		Agent   any    `json:"agent"`
	}{
		Message: "Liên kết thiết bị thành công",
		Agent:   toPCAgentResponse(agent),
	})
}

func (h *PCAgentHandler) HandleGetPairingStatus(w http.ResponseWriter, r *http.Request) {
	result, err := h.pairing.GetPairingStatus(r.Context(), domain.PairingStatusInput{
		PairingSessionID: r.URL.Query().Get("pairing_session_id"),
		DeviceCode:       r.URL.Query().Get("device_code"),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPairingSessionNotFound):
			writeMessageError(w, http.StatusBadRequest, "Mã liên kết không hợp lệ")
		case errors.Is(err, usecase.ErrPairingSessionMismatch):
			writeMessageError(w, http.StatusBadRequest, "Phiên liên kết không hợp lệ")
		default:
			writeInternalError(w, err)
		}
		return
	}

	response := map[string]any{"status": result.Status}
	if result.PCAgentID != nil {
		response["pc_agent_id"] = *result.PCAgentID
	}
	if result.AgentSecret != "" {
		response["agent_secret"] = result.AgentSecret
	}
	if result.CredentialIssued {
		response["credential_issued"] = true
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *PCAgentHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PCAgentID   string `json:"pc_agent_id"`
		AgentSecret string `json:"agent_secret"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	agent, err := h.pairing.VerifyPCAgent(r.Context(), domain.VerifyPCAgentInput{
		PCAgentID:   req.PCAgentID,
		AgentSecret: req.AgentSecret,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAgentCredentialInvalid):
			writeMessageError(w, http.StatusUnauthorized, "Thông tin xác thực thiết bị không hợp lệ")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Message          string `json:"message"`
		PCAgentID        string `json:"pc_agent_id"`
		ProtectionStatus string `json:"protection_status"`
	}{
		Message:          "agent verified",
		PCAgentID:        agent.ID,
		ProtectionStatus: agent.ProtectionStatus,
	})
}

func (h *PCAgentHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	agents, err := h.pairing.ListPCAgents(r.Context(), userID)
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

	writeJSON(w, http.StatusOK, items)
}

func (h *PCAgentHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	err := h.pairing.DeletePCAgent(r.Context(), domain.DeletePCAgentInput{
		UserID:    userID,
		PCAgentID: r.PathValue("pc_agent_id"),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPCAgentNotFound):
			writeError(w, http.StatusNotFound, "pc_agent_not_found")
		default:
			writeInternalError(w, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PCAgentHandler) HandleUpdateProtection(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	agent, err := h.pairing.UpdatePCAgentProtection(r.Context(), domain.UpdatePCAgentProtectionInput{
		UserID:    userID,
		PCAgentID: r.PathValue("pc_agent_id"),
		Enabled:   req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPCAgentNotFound):
			writeError(w, http.StatusNotFound, "pc_agent_not_found")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toPCAgentResponse(agent))
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
		OSType           string  `json:"os_type"`
		Status           string  `json:"agent_status"`
		ProtectionStatus string  `json:"protection_status"`
		LastSeenAt       *string `json:"last_seen_at"`
	}{
		ID:               agent.ID,
		DeviceName:       agent.DeviceName,
		OSType:           agent.OSType,
		Status:           agent.Status,
		ProtectionStatus: agent.ProtectionStatus,
		LastSeenAt:       lastSeenAt,
	}
}

func secondsUntil(t time.Time) int64 {
	remaining := time.Until(t)
	if remaining <= 0 {
		return 0
	}

	return int64((remaining + time.Second - time.Nanosecond) / time.Second)
}
