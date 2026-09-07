package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestReadinessCheck_NilClients(t *testing.T) {
	// If db and redis clients are nil (e.g. in test env), readiness returns healthy
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.ReadinessCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", resp["status"])
	}
}

func TestAlertWebhook(t *testing.T) {
	h := &Handler{}

	payload := AlertNotification{
		Receiver: "default-webhook",
		Status:   "firing",
		Alerts: []struct {
			Status      string            `json:"status"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			StartsAt    time.Time         `json:"startsAt"`
			EndsAt      time.Time         `json:"endsAt"`
		}{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighErrorRate5xx",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"description": "5xx error rate exceeds 1%",
				},
				StartsAt: time.Now().UTC(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.AlertWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "received" {
		t.Errorf("expected status 'received', got %v", resp["status"])
	}
	if resp["received"].(float64) != 1 {
		t.Errorf("expected received count 1, got %v", resp["received"])
	}
}
