package usecase

import (
	"context"
	"errors"
	"time"
	"wetalk/internal/entity"
	"wetalk/internal/repository"
	"wetalk/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrEmailAlreadyTaken     = errors.New("email already taken")
	ErrUsernameAlreadyTaken  = errors.New("username already taken")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrExpiredRefreshToken   = errors.New("refresh token has expired")
	ErrRevokedRefreshToken   = errors.New("refresh token has been revoked")
)

type AuthUsecase interface {
	Register(ctx context.Context, req entity.RegisterRequest) (entity.AuthResponse, error)
	Login(ctx context.Context, req entity.LoginRequest) (entity.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (entity.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAllDevices(ctx context.Context, userId string) error
	ValidateAccessToken(token string) (*entity.TokenClaims, error)
}

type authUsecase struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *jwt.JWTManager
}

func NewAuthUsecase(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtManager *jwt.JWTManager,
) AuthUsecase {
	return &authUsecase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
	}
}

func (u *authUsecase) Register(ctx context.Context, req entity.RegisterRequest) (entity.AuthResponse, error) {
	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Username == "" || req.Name == "" {
		return entity.AuthResponse{}, errors.New("all fields are required")
	}

	// Check if email already exists
	emailExists, err := u.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return entity.AuthResponse{}, err
	}
	if emailExists {
		return entity.AuthResponse{}, ErrEmailAlreadyTaken
	}

	// Check if username already exists
	usernameExists, err := u.userRepo.UsernameExists(ctx, req.Username)
	if err != nil {
		return entity.AuthResponse{}, err
	}
	if usernameExists {
		return entity.AuthResponse{}, ErrUsernameAlreadyTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Create user
	user := entity.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		IsOnline: false,
	}

	userId, err := u.userRepo.Create(ctx, user)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	user.Id = userId

	// Generate access token
	accessToken, err := u.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Generate refresh token
	refreshTokenString, err := u.jwtManager.GenerateRefreshToken()
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Store refresh token in database
	refreshToken := entity.RefreshToken{
		UserId:    userId,
		Token:     refreshTokenString,
		ExpiresAt: u.jwtManager.GetRefreshTokenExpiration(),
	}

	err = u.refreshTokenRepo.Create(ctx, refreshToken)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Remove password from response
	user.Password = ""

	return entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		User:         user,
	}, nil
}

func (u *authUsecase) Login(ctx context.Context, req entity.LoginRequest) (entity.AuthResponse, error) {
	// Get user by email
	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return entity.AuthResponse{}, ErrInvalidCredentials
		}
		return entity.AuthResponse{}, err
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return entity.AuthResponse{}, ErrInvalidCredentials
	}

	// Generate access token
	accessToken, err := u.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Generate refresh token
	refreshTokenString, err := u.jwtManager.GenerateRefreshToken()
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Store refresh token in database
	refreshToken := entity.RefreshToken{
		UserId:    user.Id,
		Token:     refreshTokenString,
		ExpiresAt: u.jwtManager.GetRefreshTokenExpiration(),
	}

	err = u.refreshTokenRepo.Create(ctx, refreshToken)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Remove password from response
	user.Password = ""

	return entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		User:         user,
	}, nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshTokenString string) (entity.AuthResponse, error) {
	// Get refresh token from database
	refreshToken, err := u.refreshTokenRepo.GetByToken(ctx, refreshTokenString)
	if err != nil {
		return entity.AuthResponse{}, ErrInvalidRefreshToken
	}

	// Check if token is revoked
	if refreshToken.IsRevoked {
		return entity.AuthResponse{}, ErrRevokedRefreshToken
	}

	// Check if token is expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return entity.AuthResponse{}, ErrExpiredRefreshToken
	}

	// Get user
	user, err := u.userRepo.Get(ctx, refreshToken.UserId)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Generate new access token
	accessToken, err := u.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Generate new refresh token (token rotation)
	newRefreshTokenString, err := u.jwtManager.GenerateRefreshToken()
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Revoke old refresh token
	err = u.refreshTokenRepo.Revoke(ctx, refreshTokenString)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Store new refresh token
	newRefreshToken := entity.RefreshToken{
		UserId:    user.Id,
		Token:     newRefreshTokenString,
		ExpiresAt: u.jwtManager.GetRefreshTokenExpiration(),
	}

	err = u.refreshTokenRepo.Create(ctx, newRefreshToken)
	if err != nil {
		return entity.AuthResponse{}, err
	}

	// Remove password from response
	user.Password = ""

	return entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenString,
		User:         user,
	}, nil
}

func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	// Revoke the refresh token
	err := u.refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		return err
	}

	return nil
}

func (u *authUsecase) LogoutAllDevices(ctx context.Context, userId string) error {
	// Revoke all refresh tokens for the user
	err := u.refreshTokenRepo.RevokeAllByUserId(ctx, userId)
	if err != nil {
		return err
	}

	return nil
}

func (u *authUsecase) ValidateAccessToken(token string) (*entity.TokenClaims, error) {
	return u.jwtManager.ValidateAccessToken(token)
}