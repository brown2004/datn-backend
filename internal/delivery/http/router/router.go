package router

import (
	"net/http"

	deliveryhttp "datn-backend/internal/delivery/http"
	"datn-backend/internal/token"
)

func NewRouter(
	authHandler *deliveryhttp.AuthHandler,
	pcAgentHandler *deliveryhttp.PCAgentHandler,
	mobileDeviceHandler *deliveryhttp.MobileDeviceHandler,
	tokenService *token.Service,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	registerAuthRoutes(mux, authHandler, tokenService)
	registerPCAgentRoutes(mux, pcAgentHandler, tokenService)
	registerMobileDeviceRoutes(mux, mobileDeviceHandler, tokenService)

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

func registerPCAgentRoutes(mux *http.ServeMux, handler *deliveryhttp.PCAgentHandler, tokenService *token.Service) {
	mux.HandleFunc("POST /api/pc-agents/pairing/start", handler.HandleStartPairing)
	mux.HandleFunc("GET /api/pc-agents/pairing/status", handler.HandleGetPairingStatus)
	mux.HandleFunc("POST /api/pc-agents/verify", handler.HandleVerify)
	mux.HandleFunc("PATCH /api/pc-agents/me/protection", handler.HandleUpdateOwnProtection)
	mux.Handle("POST /api/pc-agents/pairing/confirm", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(handler.HandleConfirmPairing)))
	mux.Handle("GET /api/pc-agents", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(handler.HandleList)))
	mux.Handle("DELETE /api/pc-agents/{pc_agent_id}", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(handler.HandleDelete)))
	mux.Handle("PATCH /api/pc-agents/{pc_agent_id}/protection", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(handler.HandleUpdateProtection)))
}

func registerMobileDeviceRoutes(mux *http.ServeMux, handler *deliveryhttp.MobileDeviceHandler, tokenService *token.Service) {
	mux.Handle("POST /api/mobile-devices", deliveryhttp.RequireAuth(tokenService, http.HandlerFunc(handler.HandleRegister)))
}
