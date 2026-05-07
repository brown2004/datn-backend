package http

import "net/http"

type AlertHandler struct{}

func NewAlertHandler() *AlertHandler {
	return &AlertHandler{}
}

func (h *AlertHandler) HandleCreateAlert(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}
