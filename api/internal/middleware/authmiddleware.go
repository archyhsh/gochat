// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/response"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "Missing Authorization Header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(w, "Invalid Token Format")
			return
		}

		claims, err := m.jwtManager.ParseToken(parts[1])
		if err != nil {
			log.Printf("AuthMiddleware: Token validation failed: %v", err)
			response.Unauthorized(w, "Invalid or Expired Token: "+err.Error())
			return
		}

		// Unified key name: user_id
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)

		next(w, r.WithContext(ctx))
	}
}
