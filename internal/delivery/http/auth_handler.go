package http

import (
	"errors"
	"net/http"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
	"datn-backend/internal/usecase"
)

type AuthHandler struct {
	auth *usecase.AuthUseCase
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) HandleRequestRegisterOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	result, err := h.auth.RequestRegisterOTP(r.Context(), domain.RequestRegisterOTPInput{
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPhoneAlreadyExists):
			writeError(w, http.StatusConflict, "phone_number_already_exists")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, struct {
		ExpiresAt string  `json:"expires_at"`
		DevOTP    *string `json:"dev_otp,omitempty"`
	}{
		ExpiresAt: result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		DevOTP:    result.DevOTP,
	})
}

func (h *AuthHandler) HandleVerifyRegisterOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
		OTP         string `json:"otp"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	verification, err := h.auth.VerifyRegisterOTP(r.Context(), domain.VerifyRegisterOTPInput{
		PhoneNumber: req.PhoneNumber,
		OTP:         req.OTP,
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
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toRegisterOTPVerificationResponse(verification))
}

func (h *AuthHandler) HandleCompleteRegister(w http.ResponseWriter, r *http.Request) {
	phoneNumber, ok := registerPhoneFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_register_phone")
		return
	}

	var req struct {
		FullName string `json:"full_name"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	session, err := h.auth.CompleteRegister(r.Context(), phoneNumber, domain.CompleteRegisterInput{
		FullName: req.FullName,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrPhoneAlreadyExists):
			writeError(w, http.StatusConflict, "phone_number_already_exists")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, toAuthSessionResponse(session))
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
		Password    string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	session, err := h.auth.Login(r.Context(), domain.LoginInput{
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toAuthSessionResponse(session))
}

func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	tokens, err := h.auth.Refresh(r.Context(), domain.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrInvalidRefreshToken), errors.Is(err, usecase.ErrRefreshTokenRevoked):
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token")
		case errors.Is(err, usecase.ErrRefreshTokenExpired):
			writeError(w, http.StatusUnauthorized, "refresh_token_expired")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toAuthTokensResponse(*tokens))
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := h.auth.Logout(r.Context(), domain.LogoutInput{RefreshToken: req.RefreshToken}); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		case errors.Is(err, usecase.ErrInvalidRefreshToken):
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) HandleLogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	if err := h.auth.LogoutAll(r.Context(), userID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input")
		default:
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) HandleLinkEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_user_id")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
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
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func toAuthSessionResponse(session *domain.AuthSession) any {
	return struct {
		User         any    `json:"user"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresAt    string `json:"expires_at"`
		ExpiresIn    int64  `json:"expires_in"`
	}{
		User:         toUserResponse(session.User),
		AccessToken:  session.Tokens.AccessToken,
		RefreshToken: session.Tokens.RefreshToken,
		TokenType:    session.Tokens.TokenType,
		ExpiresAt:    session.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		ExpiresIn:    session.Tokens.ExpiresIn,
	}
}

func toRegisterOTPVerificationResponse(verification *domain.RegisterOTPVerification) any {
	return struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresAt   string `json:"expires_at"`
		ExpiresIn   int64  `json:"expires_in"`
	}{
		AccessToken: verification.AccessToken,
		TokenType:   verification.TokenType,
		ExpiresAt:   verification.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		ExpiresIn:   verification.ExpiresIn,
	}
}

func toAuthTokensResponse(tokens domain.AuthTokens) any {
	return struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresAt    string `json:"expires_at"`
		ExpiresIn    int64  `json:"expires_in"`
	}{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		ExpiresIn:    tokens.ExpiresIn,
	}
}

func toUserResponse(user *domain.User) any {
	return struct {
		ID          string  `json:"id"`
		Email       *string `json:"email"`
		PhoneNumber string  `json:"phone_number"`
		FullName    string  `json:"full_name"`
		CreatedAt   string  `json:"created_at"`
	}{
		ID:          user.ID,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		FullName:    user.FullName,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
