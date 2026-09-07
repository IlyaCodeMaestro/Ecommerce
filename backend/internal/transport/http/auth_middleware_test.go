package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/service"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test_secret_for_middleware_testing_key_12345"
	authService := service.NewAuthService(nil, nil, secret)

	user := &domain.User{
		ID:    "usr-middleware-1",
		Email: "alex@example.com",
		Role:  domain.RoleCustomer,
	}
	token, _, err := authService.CreateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to create access token: %v", err)
	}

	handlerCalled := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		claims := GetUserClaims(r.Context())
		if claims == nil {
			t.Fatal("expected user claims in context, got nil")
		}
		if claims.UserID != user.ID {
			t.Errorf("expected user ID %s, got %s", user.ID, claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := AuthMiddleware(authService)(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be invoked")
	}
}

func TestAuthMiddleware_MissingOrInvalidToken(t *testing.T) {
	secret := "test_secret_for_middleware_testing_key_12345"
	authService := service.NewAuthService(nil, nil, secret)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AuthMiddleware(authService)(dummyHandler)

	// Case 1: Missing header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing header, got %d", rec.Code)
	}

	// Case 2: Invalid prefix
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic abcdef")
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad prefix, got %d", rec.Code)
	}

	// Case 3: Bogus token
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.signature")
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rec.Code)
	}
}

func TestRequireRole_RBAC(t *testing.T) {
	secret := "test_secret_for_middleware_testing_key_12345"
	authService := service.NewAuthService(nil, nil, secret)

	adminUser := &domain.User{ID: "adm-1", Email: "admin@store.local", Role: domain.RoleAdmin}
	customerUser := &domain.User{ID: "cus-1", Email: "buyer@store.local", Role: domain.RoleCustomer}

	adminToken, _, _ := authService.CreateAccessToken(adminUser)
	customerToken, _, _ := authService.CreateAccessToken(customerUser)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := AuthMiddleware(authService)(RequireRole(domain.RoleAdmin)(dummyHandler))

	// Admin access granted
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", rec.Code)
	}

	// Customer access forbidden
	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+customerToken)
	rec = httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for customer on admin route, got %d", rec.Code)
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	secret := "test_secret_for_middleware_testing_key_12345"
	authService := service.NewAuthService(nil, nil, secret)

	user := &domain.User{ID: "usr-opt-1", Email: "guest_buyer@local", Role: domain.RoleCustomer}
	token, _, _ := authService.CreateAccessToken(user)

	var receivedUserID string
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims := GetUserClaims(r.Context()); claims != nil {
			receivedUserID = claims.UserID
		} else {
			receivedUserID = "anon"
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := OptionalAuthMiddleware(authService)(dummyHandler)

	// Authenticated request
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || receivedUserID != "usr-opt-1" {
		t.Errorf("expected usr-opt-1, got %s with status %d", receivedUserID, rec.Code)
	}

	// Anonymous request (no Authorization header)
	req = httptest.NewRequest(http.MethodPost, "/checkout", nil)
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || receivedUserID != "anon" {
		t.Errorf("expected anon, got %s with status %d", receivedUserID, rec.Code)
	}
}
