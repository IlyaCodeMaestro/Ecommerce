package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	productService *service.ProductService
	orderService   *service.OrderService
	authService    *service.AuthService
	cartService    *service.CartService
	redisClient    *redis.Client
	db             *postgres.DB
}

func NewHandler(
	productService *service.ProductService,
	orderService *service.OrderService,
	authService *service.AuthService,
	cartService *service.CartService,
	redisClient *redis.Client,
	db *postgres.DB,
) *Handler {
	return &Handler{
		productService: productService,
		orderService:   orderService,
		authService:    authService,
		cartService:    cartService,
		redisClient:    redisClient,
		db:             db,
	}
}

// HealthCheck serves Kubernetes liveness probes
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.2.0-production",
	})
}

// ReadinessCheck serves Kubernetes readiness probes by verifying PostgreSQL and Redis connections
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{
		"postgres": "up",
		"redis":    "up",
	}
	allHealthy := true

	if h.db != nil && h.db.Pool != nil {
		if err := h.db.Pool.Ping(ctx); err != nil {
			checks["postgres"] = fmt.Sprintf("down: %v", err)
			allHealthy = false
		}
	}

	if h.redisClient != nil && h.redisClient.RDB != nil {
		if err := h.redisClient.RDB.Ping(ctx).Err(); err != nil {
			checks["redis"] = fmt.Sprintf("down: %v", err)
			allHealthy = false
		}
	}

	status := http.StatusOK
	respStatus := "healthy"
	if !allHealthy {
		status = http.StatusServiceUnavailable
		respStatus = "degraded"
	}

	respondJSON(w, status, map[string]interface{}{
		"status":     respStatus,
		"timestamp":  time.Now().UTC(),
		"components": checks,
	})
}

type AlertNotification struct {
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Alerts   []struct {
		Status      string            `json:"status"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		StartsAt    time.Time         `json:"startsAt"`
		EndsAt      time.Time         `json:"endsAt"`
	} `json:"alerts"`
}

// AlertWebhook handles incoming alerts pushed from Prometheus Alertmanager
func (h *Handler) AlertWebhook(w http.ResponseWriter, r *http.Request) {
	var notification AlertNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		respondError(w, http.StatusBadRequest, "invalid alertmanager payload")
		return
	}

	for _, alert := range notification.Alerts {
		log.Printf("[ALERTMANAGER] Alert: %s | Severity: %s | Status: %s | Description: %s",
			alert.Labels["alertname"],
			alert.Labels["severity"],
			alert.Status,
			alert.Annotations["description"],
		)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "received",
		"received": len(notification.Alerts),
	})
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	category := q.Get("category")
	sort := q.Get("sort")

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	var minPrice, maxPrice *float64
	if minStr := q.Get("min_price"); minStr != "" {
		if val, err := strconv.ParseFloat(minStr, 64); err == nil {
			minPrice = &val
		}
	}
	if maxStr := q.Get("max_price"); maxStr != "" {
		if val, err := strconv.ParseFloat(maxStr, 64); err == nil {
			maxPrice = &val
		}
	}

	filter := domain.ProductFilter{
		Query:    query,
		Category: category,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		SortBy:   sort,
		Limit:    limit,
		Offset:   offset,
	}

	result, err := h.productService.SearchProducts(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to search products")
		return
	}

	// Browser cache headers for client-side speed
	w.Header().Set("Cache-Control", "public, max-age=5")
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.productService.GetProductByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch product")
		return
	}
	if product == nil {
		respondError(w, http.StatusNotFound, "product not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=10")
	respondJSON(w, http.StatusOK, product)
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.productService.GetCategories(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch categories")
		return
	}
	respondJSON(w, http.StatusOK, cats)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		req.UserID = "anon-user"
		if claims := GetUserClaims(r.Context()); claims != nil && claims.UserID != "" {
			req.UserID = claims.UserID
		} else {
			req.UserID = "anon-user"
		}
	}

	// Extract Idempotency-Key from HTTP header if not in body
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}

	resp, err := h.orderService.CreateOrderAsync(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrOrderInProgress) {
			respondError(w, http.StatusConflict, "order is currently being processed, please wait")
			return
		}
		if errors.Is(err, service.ErrInsufficientStock) {
			respondError(w, http.StatusConflict, "item is out of stock")
			return
		}
		if errors.Is(err, service.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, "product does not exist")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 202 Accepted: async order event published to Kafka
	// 202 Accepted: async order event published to Kafka & persisted in Outbox
	respondJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	order, err := h.orderService.GetOrderByID(r.Context(), orderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch order")
		return
	}
	if order == nil {
		respondError(w, http.StatusNotFound, "order not found (may still be syncing from queue)")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// CancelOrder allows a customer or admin to cancel an order and trigger compensating restock
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var req domain.CancelOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional reason

	claims := GetUserClaims(r.Context())
	userID := ""
	userRole := ""
	if claims != nil {
		userID = claims.UserID
		userRole = claims.Role
	}

	order, err := h.orderService.CancelOrder(r.Context(), orderID, userID, userRole, req.Reason)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondError(w, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			respondError(w, http.StatusForbidden, "you are not authorized to cancel this order")
			return
		}
		if errors.Is(err, service.ErrInvalidStateTransition) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "cancelled",
		"message": "Order successfully cancelled and inventory restocked",
		"order":   order,
	})
}

// ListUserOrders returns order history for authenticated customer
func (h *Handler) ListUserOrders(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r.Context())
	if claims == nil || claims.UserID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	orders, total, err := h.orderService.ListUserOrders(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query order history")
		return
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	respondJSON(w, http.StatusOK, domain.OrderListResponse{
		Orders: orders,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// AdminListOrders returns all store orders for administrators with optional status filter
func (h *Handler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	statusStr := q.Get("status")

	var statusFilter *domain.OrderStatus
	if statusStr != "" {
		s := domain.OrderStatus(statusStr)
		statusFilter = &s
	}

	orders, total, err := h.orderService.ListAllOrders(r.Context(), statusFilter, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query admin orders")
		return
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	respondJSON(w, http.StatusOK, domain.OrderListResponse{
		Orders: orders,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// AdminUpdateOrderStatus advances order lifecycle state with FSM validation
func (h *Handler) AdminUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var req domain.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status == "" {
		respondError(w, http.StatusBadRequest, "status is required")
		return
	}

	order, err := h.orderService.UpdateOrderStatus(r.Context(), orderID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondError(w, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, service.ErrInvalidStateTransition) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  order.Status,
		"message": "Order status updated successfully",
		"order":   order,
	})
}

// StreamOrderStatus streams real-time Server-Sent Events (SSE) for order lifecycle transitions
func (h *Handler) StreamOrderStatus(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, "Missing order id", http.StatusBadRequest)
		return
	}

	// SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 1. Initial State: Enqueued in Kafka
	fmt.Fprintf(w, "data: {\"order_id\":\"%s\",\"status\":\"ACCEPTED\",\"step\":1,\"message\":\"Dispatched to Kafka queue\",\"timestamp\":\"%s\"}\n\n", orderID, time.Now().UTC().Format(time.RFC3339))
	flusher.Flush()

	// 2. Check if order was already persisted by worker in Postgres
	existing, _ := h.orderService.GetOrderByID(r.Context(), orderID)
	if existing != nil {
		fmt.Fprintf(w, "data: {\"order_id\":\"%s\",\"status\":\"COMPLETED\",\"step\":3,\"message\":\"Order verified in PostgreSQL\",\"timestamp\":\"%s\"}\n\n", orderID, time.Now().UTC().Format(time.RFC3339))
		flusher.Flush()
		return
	}

	// Intermediate step notification
	time.Sleep(150 * time.Millisecond)
	fmt.Fprintf(w, "data: {\"order_id\":\"%s\",\"status\":\"PROCESSING\",\"step\":2,\"message\":\"Worker batch processing...\",\"timestamp\":\"%s\"}\n\n", orderID, time.Now().UTC().Format(time.RFC3339))
	flusher.Flush()

	// 3. Subscribe to Redis Pub/Sub channel for live completion notification
	if h.redisClient != nil {
		channelName := fmt.Sprintf("order:%s:status", orderID)
		pubsub := h.redisClient.Subscribe(r.Context(), channelName)
		defer pubsub.Close()

		ch := pubsub.Channel()
		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-r.Context().Done():
				return
			case <-timeout:
				// Fallback timeout check against DB
				finalCheck, _ := h.orderService.GetOrderByID(r.Context(), orderID)
				if finalCheck != nil {
					fmt.Fprintf(w, "data: {\"order_id\":\"%s\",\"status\":\"COMPLETED\",\"step\":3,\"message\":\"Order completed in PostgreSQL\",\"timestamp\":\"%s\"}\n\n", orderID, time.Now().UTC().Format(time.RFC3339))
					flusher.Flush()
				}
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
				flusher.Flush()
				return
			}
		}
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// PaymentWebhook handles incoming payment events from payment gateways (e.g., Stripe/ЮKassa)
// with HMAC-SHA256 signature verification.
func (h *Handler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		signature = r.Header.Get("Stripe-Signature")
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var payload domain.PaymentWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	if err := h.orderService.ProcessPaymentWebhook(r.Context(), signature, bodyBytes, payload); err != nil {
		if errors.Is(err, service.ErrInvalidSignature) {
			respondError(w, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
		if errors.Is(err, service.ErrOrderNotFound) {
			respondError(w, http.StatusNotFound, "order not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"received": true,
		"order_id": payload.OrderID,
	})
}

// SimulatePayment is a testing & demo endpoint that generates a valid HMAC signature
// and processes a simulated payment webhook for the given order.
func (h *Handler) SimulatePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	payload := domain.PaymentWebhookPayload{
		Event:     "payment.succeeded",
		PaymentID: fmt.Sprintf("pay_%d", time.Now().UnixNano()),
		OrderID:   req.OrderID,
		Amount:    req.Amount,
		Currency:  "USD",
	}

	payloadBytes, _ := json.Marshal(payload)
	validSig := h.orderService.GenerateWebhookSignature(payloadBytes)

	if err := h.orderService.ProcessPaymentWebhook(r.Context(), validSig, payloadBytes, payload); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "PAID",
		"order_id":   req.OrderID,
		"payment_id": payload.PaymentID,
		"signature":  validSig,
		"message":    "Payment successfully verified and order completed via HMAC-SHA256",
	})
}
