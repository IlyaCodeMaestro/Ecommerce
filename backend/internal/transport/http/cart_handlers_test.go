package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/service"
)

func TestResolveCartKey(t *testing.T) {
	// Case 1: Authenticated user
	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	claims := &domain.UserClaims{UserID: "usr-123", Email: "user@test.local"}
	ctx := context.WithValue(req.Context(), UserClaimsContextKey, claims)
	req = req.WithContext(ctx)

	key := resolveCartKey(req)
	if key != "usr:usr-123" {
		t.Errorf("expected usr:usr-123, got %s", key)
	}

	// Case 2: Guest with X-Session-ID
	req = httptest.NewRequest(http.MethodGet, "/cart", nil)
	req.Header.Set("X-Session-ID", "sess-xyz-987")
	key = resolveCartKey(req)
	if key != "guest:sess-xyz-987" {
		t.Errorf("expected guest:sess-xyz-987, got %s", key)
	}

	// Case 3: Anonymous fallback
	req = httptest.NewRequest(http.MethodGet, "/cart", nil)
	key = resolveCartKey(req)
	if key != "guest:anonymous" {
		t.Errorf("expected guest:anonymous, got %s", key)
	}
}

func TestCartHandlers_GetCartEmpty(t *testing.T) {
	cartService := service.NewCartService(nil, nil)
	h := &Handler{cartService: cartService}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	req.Header.Set("X-Session-ID", "session-test-01")
	rec := httptest.NewRecorder()

	h.GetCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

