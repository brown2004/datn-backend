package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
	"datn-backend/internal/usecase"
)

type AuthHandler struct {
	auth *usecase.AuthUseCase
}

type requestRegisterOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type requestRegisterOTPResponse struct {
	ExpiresAt string  `json:"expires_at"`
	DevOTP    *string `json:"dev_otp,omitempty"`
}

type verifyRegisterRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
	FullName    string `json:"full_name"`
	Password    string `json:"password"`
}

type linkEmailRequest struct {
	Email string `json:"email"`
}

type userResponse struct {
	ID          string  `json:"id"`
	Email       *string `json:"email"`
	PhoneNumber string  `json:"phone_number"`
	FullName    string  `json:"full_name"`
	CreatedAt   string  `json:"created_at"`
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) HandleRequestRegisterOTP(w http.ResponseWriter, r *http.Request) {
	var req requestRegisterOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	result, err := h.auth.RequestRegisterOTP(r.Context(), usecase.RequestRegisterOTPInput{
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPhoneAlreadyExists):
			writeError(w, http.StatusConflict, "phone_number_already_exists")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusOK, requestRegisterOTPResponse{
		ExpiresAt: result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		DevOTP:    result.DevOTP,
	})
}

func (h *AuthHandler) HandleVerifyRegister(w http.ResponseWriter, r *http.Request) {
	var req verifyRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	user, err := h.auth.VerifyRegister(r.Context(), usecase.VerifyRegisterInput{
		PhoneNumber: req.PhoneNumber,
		OTP:         req.OTP,
		FullName:    req.FullName,
		Password:    req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPhoneAlreadyExists):
			writeError(w, http.StatusConflict, "phone_number_already_exists")
		case errors.Is(err, repo.ErrOTPNotFound):
			writeError(w, http.StatusBadRequest, "otp_not_found")
		case errors.Is(err, usecase.ErrOTPExpired):
			writeError(w, http.StatusBadRequest, "otp_expired")
		case errors.Is(err, usecase.ErrOTPUsed):
			writeError(w, http.StatusBadRequest, "otp_already_used")
		case errors.Is(err, usecase.ErrOTPTooManyAttempts):
			writeError(w, http.StatusTooManyRequests, "otp_too_many_attempts")
		case errors.Is(err, usecase.ErrInvalidOTP):
			writeError(w, http.StatusBadRequest, "invalid_otp")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

func (h *AuthHandler) HandleLinkEmail(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req linkEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	user, err := h.auth.LinkEmail(r.Context(), userID, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email_already_exists")
		case errors.Is(err, repo.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user_not_found")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func toUserResponse(user *domain.User) userResponse {
	return userResponse{
		ID:          user.ID,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		FullName:    user.FullName,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
