package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"wetalk/internal/entity"
	"wetalk/internal/usecase"
)

type AuthHandler struct {
	authUc usecase.AuthUsecase
}

func NewAuthHandler(authUc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUc: authUc,
	}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req entity.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{
			Message: "invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Basic validation
	if req.Email == "" || req.Password == "" || req.Username == "" || req.Name == "" {
		response := Response{
			Message: "email, username, password, and name are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate password length
	if len(req.Password) < 6 {
		response := Response{
			Message: "password must be at least 6 characters",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate username length
	if len(req.Username) < 3 {
		response := Response{
			Message: "username must be at least 3 characters",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	authResponse, err := h.authUc.Register(r.Context(), req)
	if err != nil {
		log.Printf("Register error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "internal server error"

		switch err {
		case usecase.ErrEmailAlreadyTaken:
			statusCode = http.StatusConflict
			message = "email already taken"
		case usecase.ErrUsernameAlreadyTaken:
			statusCode = http.StatusConflict
			message = "username already taken"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set refresh token as HttpOnly cookie
	h.setRefreshTokenCookie(w, authResponse.RefreshToken)

	// Don't send refresh token in JSON response (it's in cookie)
	authResponse.RefreshToken = ""

	response := Response{
		Message: "registration successful",
		Data:    authResponse,
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req entity.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{Message: "invalid request body"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Email == "" || req.Password == "" {
		response := Response{Message: "email and password are required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	authResponse, err := h.authUc.Login(r.Context(), req)
	if err != nil {
		log.Printf("Login error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "internal server error"

		if err == usecase.ErrInvalidCredentials {
			statusCode = http.StatusUnauthorized
			message = "invalid email or password"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set refresh token as HttpOnly cookie
	h.setRefreshTokenCookie(w, authResponse.RefreshToken)

	// Don't send refresh token in JSON response (it's in cookie)
	authResponse.RefreshToken = ""

	response := Response{
		Message: "login successful",
		Data:    authResponse,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Try to get refresh token from cookie first
	refreshToken := ""
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		refreshToken = cookie.Value
	}

	// If not in cookie, try to get from request body
	if refreshToken == "" {
		var req entity.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		response := Response{Message: "refresh token is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	authResponse, err := h.authUc.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("Refresh token error: %v", err)

		statusCode := http.StatusUnauthorized
		message := "invalid or expired refresh token"

		switch err {
		case usecase.ErrInvalidRefreshToken:
			message = "invalid refresh token"
		case usecase.ErrExpiredRefreshToken:
			message = "refresh token has expired"
		case usecase.ErrRevokedRefreshToken:
			message = "refresh token has been revoked"
		}

		// Clear the invalid cookie
		h.clearRefreshTokenCookie(w)

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set new refresh token as HttpOnly cookie
	h.setRefreshTokenCookie(w, authResponse.RefreshToken)

	// Don't send refresh token in JSON response (it's in cookie)
	authResponse.RefreshToken = ""

	response := Response{
		Message: "token refreshed successfully",
		Data:    authResponse,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Try to get refresh token from cookie first
	refreshToken := ""
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		refreshToken = cookie.Value
	}

	// If not in cookie, try to get from request body
	if refreshToken == "" {
		var req entity.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" {
		err := h.authUc.Logout(r.Context(), refreshToken)
		if err != nil {
			log.Printf("Logout error: %v", err)
		}
	}

	// Clear the cookie
	h.clearRefreshTokenCookie(w)

	response := Response{
		Message: "logout successful",
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /auth/logout-all
func (h *AuthHandler) LogoutAllDevices(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.authUc.LogoutAllDevices(r.Context(), userClaims.UserId)
	if err != nil {
		log.Printf("Logout all devices error: %v", err)
		response := Response{Message: "internal server error"}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Clear the cookie
	h.clearRefreshTokenCookie(w)

	response := Response{
		Message: "logged out from all devices successfully",
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to set refresh token cookie
func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,                    // Cannot be accessed by JavaScript
		Secure:   false,                   // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,    // CSRF protection
		MaxAge:   30 * 24 * 60 * 60,       // 30 days
	}
	http.SetCookie(w, cookie)
}

// Helper function to clear refresh token cookie
func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)
}