package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"ecommerce-backend/internal/domain"
)

func TestVerifySignature(t *testing.T) {
	secret := "test_webhook_secret_key_123"
	s := NewOrderService(nil, nil, nil, nil, nil, nil, nil, secret)

	payload := []byte(`{"event":"payment.succeeded","order_id":"test-123","amount":99.99}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	if !s.verifySignature(payload, validSig) {
		t.Errorf("expected valid signature to verify successfully")
	}

	// Test invalid signature
	if s.verifySignature(payload, "invalid_hex_signature") {
		t.Errorf("expected invalid signature to fail verification")
	}

	// Test empty signature
	if s.verifySignature(payload, "") {
		t.Errorf("expected empty signature to fail verification")
	}

	// Test altered payload
	alteredPayload := []byte(`{"event":"payment.succeeded","order_id":"test-123","amount":1000.00}`)
	if s.verifySignature(alteredPayload, validSig) {
		t.Errorf("expected altered payload to fail HMAC verification")
	}
}

func TestGenerateWebhookSignature(t *testing.T) {
	secret := "whsec_super_secret"
	s := NewOrderService(nil, nil, nil, nil, nil, nil, nil, secret)

	payload := []byte(`{"test":"value"}`)
	sig := s.GenerateWebhookSignature(payload)

	if !s.verifySignature(payload, sig) {
		t.Errorf("expected generated signature to verify against own service")
	}
}

func TestCreateOrderValidation(t *testing.T) {
	s := NewOrderService(nil, nil, nil, nil, nil, nil, nil, "secret")

	// Empty items
	_, err := s.CreateOrderAsync(context.Background(), domain.CreateOrderRequest{
		UserID: "user-1",
		Items:  nil,
	})
	if err != ErrInvalidRequest {
		t.Errorf("expected ErrInvalidRequest for empty items, got %v", err)
	}
}
