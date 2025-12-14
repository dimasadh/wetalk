package entity

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"` // Only in JSON response, not cookie
	User         User   `json:"user"`
}

type TokenClaims struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}