package http

import (
	"errors"
	"net/http"

	"datn-backend/internal/domain"
	"datn-backend/internal/usecase"
)

type AlertHandler struct {
	alerts *usecase.AlertUseCase
}

func NewAlertHandler(alerts *usecase.AlertUseCase) *AlertHandler {
	return &AlertHandler{alerts: alerts}
}

func (h *AlertHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	alerts, err := h.alerts.ListAlerts(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		default:
			writeInternalError(w, err)
		}
		return
	}

	items := make([]any, 0, len(alerts))
	for i := range alerts {
		items = append(items, toAlertResponse(&alerts[i]))
	}

	writeJSON(w, http.StatusOK, items)
}

func toAlertResponse(alert *domain.Alert) any {
	return struct {
		ID          string `json:"id"`
		PCAgentID   string `json:"pc_agent_id"`
		PCName      string `json:"pc_name"`
		AlertType   string `json:"alert_type"`
		Message     string `json:"message"`
		TriggeredAt string `json:"triggered_at"`
	}{
		ID:          alert.ID,
		PCAgentID:   alert.AgentID,
		PCName:      alert.AgentName,
		AlertType:   alert.Type,
		Message:     alert.Message,
		TriggeredAt: alert.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
