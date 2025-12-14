package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"wetalk/internal/usecase"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	authUc usecase.AuthUsecase
}

func NewAuthMiddleware(authUc usecase.AuthUsecase) *AuthMiddleware {
	return &AuthMiddleware{
		authUc: authUc,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response := Response{Message: "authorization header required"}
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response := Response{Message: "invalid authorization header format"}
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		token := parts[1]
		claims, err := m.authUc.ValidateAccessToken(token)
		if err != nil {
			response := Response{Message: "invalid or expired token"}
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Add user claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}