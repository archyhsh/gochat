package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/response"
)

// AuthMiddleware JWT
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w, "invalid authorization format")
				return
			}

			token := parts[1]
			claims, err := jwtManager.ParseToken(token)
			if err != nil {
				response.Unauthorized(w, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
