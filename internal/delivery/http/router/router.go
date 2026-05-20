package router

import (
	"net/http"

	deliveryhttp "datn-backend/internal/delivery/http"
	"datn-backend/internal/token"
)

func NewRouter(authHandler *deliveryhttp.AuthHandler, featureHandler *deliveryhttp.FeatureHandler, tokenService *token.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	registerAuthRoutes(mux, authHandler, tokenService)
	registerFeatureRoutes(mux, featureHandler, tokenService)

	return mux
}

func registerAuthRoutes(mux *http.ServeMux, authHandler *deliveryhttp.AuthHandler, tokenService *token.Service) {
	mux.HandleFunc("POST /api/auth/register/request-otp", authHandler.HandleRequestRegisterOTP)
	mux.HandleFunc("POST /api/auth/register/verify-otp", authHandler.HandleVerifyRegisterOTP)
	mux.Handle("POST /api/auth/register/complete", deliveryhttp.RequireRegisterAuth(tokenService, http.HandlerFunc(authHandler.HandleCompleteRegister)))
	mux.HandleFunc("POST /api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("POST /api/auth/logout", authHandler.HandleLogout)
	mux.Handle("POST /api/auth/logout-all", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(authHandler.HandleLogoutAll)))
	mux.Handle("POST /api/auth/link-email", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(authHandler.HandleLinkEmail)))
}

func registerFeatureRoutes(mux *http.ServeMux, featureHandler *deliveryhttp.FeatureHandler, tokenService *token.Service) {
	mux.Handle("POST /api/features/devices/mobile", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(featureHandler.HandleRegisterMobileDevice)))
	mux.Handle("POST /api/features/devices/pc-agents/link", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(featureHandler.HandleLinkPCAgent)))
	mux.Handle("GET /api/features/devices/pc-agents", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(featureHandler.HandleListPCAgents)))
}
