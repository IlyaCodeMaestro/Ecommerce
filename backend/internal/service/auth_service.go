package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/pkg/metrics"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	userRepo        *postgres.UserRepo
	redisClient     *redis.Client
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthService(
	userRepo *postgres.UserRepo,
	redisClient *redis.Client,
	jwtSecret string,
) *AuthService {
	if jwtSecret == "" {
		jwtSecret = "highload_super_jwt_secret_signing_key_2026"
	}
	return &AuthService{
		userRepo:        userRepo,
		redisClient:     redisClient,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || len(req.Password) < 6 {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, errors.New("valid email and password of at least 6 characters required")
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	role := domain.RoleCustomer
	if req.Role == domain.RoleAdmin {
		role = domain.RoleAdmin
	}

	userID := fmt.Sprintf("usr-%s", uuid.New().String()[:12])
	user := domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	resp, err := s.generateTokenPair(ctx, &user)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("register", "failure").Inc()
		return nil, err
	}

	metrics.AuthAttemptsTotal.WithLabelValues("register", "success").Inc()
	return resp, nil
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("login", "failure").Inc()
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	if user == nil {
		metrics.AuthAttemptsTotal.WithLabelValues("login", "failure").Inc()
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("login", "failure").Inc()
		return nil, ErrInvalidCredentials
	}

	resp, err := s.generateTokenPair(ctx, user)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("login", "failure").Inc()
		return nil, err
	}

	metrics.AuthAttemptsTotal.WithLabelValues("login", "success").Inc()
	return resp, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, oldRefreshToken string) (*domain.AuthResponse, error) {
	if strings.TrimSpace(oldRefreshToken) == "" {
		metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
		return nil, ErrInvalidToken
	}

	newRefreshToken, err := generateRandomToken(32)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	userID := ""
	if s.redisClient != nil {
		userID, err = s.redisClient.RotateRefreshToken(ctx, oldRefreshToken, newRefreshToken, s.refreshTokenTTL)
		if err != nil || userID == "" {
			metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
			return nil, ErrInvalidToken
		}
	} else {
		metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
		return nil, errors.New("redis unavailable for token rotation")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
		return nil, ErrUserNotFound
	}

	accessToken, expiresIn, err := s.CreateAccessToken(user)
	if err != nil {
		metrics.AuthAttemptsTotal.WithLabelValues("refresh", "failure").Inc()
		return nil, err
	}

	metrics.AuthAttemptsTotal.WithLabelValues("refresh", "success").Inc()
	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         user.ToResponse(),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if s.redisClient != nil && refreshToken != "" {
		return s.redisClient.RevokeRefreshToken(ctx, refreshToken)
	}
	return nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*domain.UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &domain.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*domain.UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) generateTokenPair(ctx context.Context, user *domain.User) (*domain.AuthResponse, error) {
	accessToken, expiresIn, err := s.CreateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if s.redisClient != nil {
		if err := s.redisClient.StoreRefreshToken(ctx, refreshToken, user.ID, s.refreshTokenTTL); err != nil {
			return nil, fmt.Errorf("failed to store refresh token in redis: %w", err)
		}
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		User:         user.ToResponse(),
	}, nil
}

func (s *AuthService) CreateAccessToken(user *domain.User) (string, int64, error) {
	expiresInSec := int64(s.accessTokenTTL.Seconds())
	claims := domain.UserClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "ecommerce-auth-service",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(s.accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign jwt: %w", err)
	}

	return tokenString, expiresInSec, nil
}

func generateRandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
