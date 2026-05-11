package http

import (
	"context"
	"net/http"
	"strings"

	"datn-backend/internal/token"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func requireAuth(tokenService *token.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || strings.TrimSpace(rawToken) == "" {
			writeError(w, http.StatusUnauthorized, "missing_access_token")
			return
		}

		claims, err := tokenService.VerifyAccessToken(strings.TrimSpace(rawToken))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_access_token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok && userID != ""
}
