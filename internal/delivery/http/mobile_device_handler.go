package http

import (
	"errors"
	"net/http"

	"datn-backend/internal/domain"
	"datn-backend/internal/usecase"
)

type MobileDeviceHandler struct {
	pairing *usecase.PairingUseCase
}

func NewMobileDeviceHandler(pairing *usecase.PairingUseCase) *MobileDeviceHandler {
	return &MobileDeviceHandler{pairing: pairing}
}

func (h *MobileDeviceHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
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

	device, err := h.pairing.RegisterMobileDevice(r.Context(), domain.RegisterMobileDeviceInput{
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

func toMobileDeviceResponse(device *domain.MobileDevice) any {
	return struct {
		ID       string `json:"id"`
		Platform string `json:"platform"`
	}{
		ID:       device.ID,
		Platform: device.Platform,
	}
}
