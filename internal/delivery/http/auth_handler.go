package http

import (
	"encoding/json"
	"errors"
	"log"
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

type loginRequest struct {
	PhoneNumber string `json:"phone_number"`
	Password    string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type linkEmailRequest struct {
	Email string `json:"email"`
}

type authSessionResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresAt    string       `json:"expires_at"`
	ExpiresIn    int64        `json:"expires_in"`
}

type authTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    string `json:"expires_at"`
	ExpiresIn    int64  `json:"expires_in"`
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
			writeInternalError(w, err)
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

	session, err := h.auth.VerifyRegister(r.Context(), usecase.VerifyRegisterInput{
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
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, toAuthSessionResponse(session))
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	session, err := h.auth.Login(r.Context(), usecase.LoginInput{
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
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	tokens, err := h.auth.Refresh(r.Context(), usecase.RefreshInput{
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
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	if err := h.auth.Logout(r.Context(), usecase.LogoutInput{RefreshToken: req.RefreshToken}); err != nil {
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
			writeInternalError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func toAuthSessionResponse(session *usecase.AuthSessionOutput) authSessionResponse {
	return authSessionResponse{
		User:         toUserResponse(session.User),
		AccessToken:  session.Tokens.AccessToken,
		RefreshToken: session.Tokens.RefreshToken,
		TokenType:    session.Tokens.TokenType,
		ExpiresAt:    session.Tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		ExpiresIn:    session.Tokens.ExpiresIn,
	}
}

func toAuthTokensResponse(tokens usecase.AuthTokensOutput) authTokensResponse {
	return authTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		ExpiresIn:    tokens.ExpiresIn,
	}
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

func writeInternalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal_error")
}
