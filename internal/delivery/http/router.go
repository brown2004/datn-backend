package http

import (
	"encoding/json"
	"net/http"

	"datn-backend/internal/token"
)

func NewRouter(authHandler *AuthHandler, tokenService *token.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/auth/register/request-otp", authHandler.HandleRequestRegisterOTP)
	mux.HandleFunc("POST /api/auth/register/verify", authHandler.HandleVerifyRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("POST /api/auth/logout", authHandler.HandleLogout)
	mux.Handle("POST /api/auth/logout-all", requireAuth(tokenService, http.HandlerFunc(authHandler.HandleLogoutAll)))
	mux.Handle("POST /api/auth/link-email", requireAuth(tokenService, http.HandlerFunc(authHandler.HandleLinkEmail)))

	return mux
}
