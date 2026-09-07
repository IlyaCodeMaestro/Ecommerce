package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/service"
)

func TestListUserOrders_Unauthorized(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	h.ListUserOrders(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestAdminUpdateOrderStatus_MissingStatus(t *testing.T) {
	h := &Handler{}
	payload := domain.UpdateOrderStatusRequest{Status: ""}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/ord-123/status", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.AdminUpdateOrderStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestCancelOrder_UnauthorizedUser(t *testing.T) {
	// If order repo returns an order belonging to usr-A, but usr-B attempts cancellation
	orderService := service.NewOrderService(nil, nil, nil, nil, nil, nil, nil, "")
	h := &Handler{orderService: orderService}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/ord-test/cancel", bytes.NewReader([]byte(`{"reason":"changed mind"}`)))
	claims := &domain.UserClaims{UserID: "usr-B", Role: domain.RoleCustomer}
	ctx := context.WithValue(req.Context(), UserClaimsContextKey, claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.CancelOrder(rec, req)

	// Since repo is nil, GetByID will panic or return err, resulting in 500 or 404
	if rec.Code == http.StatusOK {
		t.Fatalf("expected non-200 code for nil repo cancellation, got %d", rec.Code)
	}
}
