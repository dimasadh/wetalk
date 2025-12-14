package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
	"wetalk/internal/entity"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey              string
	accessTokenDuration    time.Duration
	refreshTokenDuration   time.Duration
}

func NewJWTManager(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:            secretKey,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// GenerateAccessToken generates a short-lived access token
func (m *JWTManager) GenerateAccessToken(user entity.User) (string, error) {
	claims := Claims{
		UserId:   user.Id,
		Email:    user.Email,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// GenerateRefreshToken generates a long-lived refresh token (cryptographically secure random string)
func (m *JWTManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetRefreshTokenExpiration returns when the refresh token should expire
func (m *JWTManager) GetRefreshTokenExpiration() time.Time {
	return time.Now().Add(m.refreshTokenDuration)
}

// ValidateAccessToken validates and parses an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*entity.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return &entity.TokenClaims{
		UserId:   claims.UserId,
		Email:    claims.Email,
		Username: claims.Username,
	}, nil
}