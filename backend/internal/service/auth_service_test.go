package service

import (
	"testing"
	"time"

	"ecommerce-backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestJWTTokenGenerationAndValidation(t *testing.T) {
	secret := "test_secret_signing_key_at_least_32_bytes_long"
	authService := NewAuthService(nil, nil, secret)

	user := &domain.User{
		ID:        "usr-test-01",
		Email:     "developer@test.local",
		Role:      domain.RoleAdmin,
		FirstName: "Jane",
		LastName:  "Doe",
	}

	tokenString, expiresIn, err := authService.CreateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to create jwt: %v", err)
	}

	if tokenString == "" {
		t.Fatal("expected non-empty token string")
	}

	if expiresIn != int64(15*time.Minute.Seconds()) {
		t.Errorf("expected 900s expiry, got %d", expiresIn)
	}

	// Validate valid token
	claims, err := authService.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("expected valid token to pass validation, got: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, claims.Email)
	}
	if claims.Role != domain.RoleAdmin {
		t.Errorf("expected role %s, got %s", domain.RoleAdmin, claims.Role)
	}
}

func TestJWTInvalidAndTamperedTokens(t *testing.T) {
	authService := NewAuthService(nil, nil, "valid_secret_key_123456789012345678")

	// Test malformed token
	_, err := authService.ValidateAccessToken("not.a.valid.jwt.token")
	if err == nil {
		t.Error("expected error for malformed token")
	}

	// Test token signed with different secret
	otherService := NewAuthService(nil, nil, "different_secret_key_98765432101234")
	user := &domain.User{ID: "usr-1", Email: "hacker@test.local", Role: domain.RoleCustomer}
	tamperedToken, _, _ := otherService.CreateAccessToken(user)

	_, err = authService.ValidateAccessToken(tamperedToken)
	if err == nil {
		t.Error("expected error for token signed with different secret")
	}
}

func TestPasswordHashing(t *testing.T) {
	rawPassword := "SecurePassword123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	// Correct password matches
	if err := bcrypt.CompareHashAndPassword(hash, []byte(rawPassword)); err != nil {
		t.Errorf("expected correct password to match hash: %v", err)
	}

	// Wrong password fails
	if err := bcrypt.CompareHashAndPassword(hash, []byte("WrongPassword123!")); err == nil {
		t.Error("expected incorrect password to fail hash comparison")
	}
}
