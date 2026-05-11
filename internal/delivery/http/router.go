package http

import (
	"encoding/json"
	"net/http"
)

func NewRouter(authHandler *AuthHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/auth/register/request-otp", authHandler.HandleRequestRegisterOTP)
	mux.HandleFunc("POST /api/auth/register/verify", authHandler.HandleVerifyRegister)
	mux.HandleFunc("POST /api/auth/link-email", authHandler.HandleLinkEmail)

	return mux
}
