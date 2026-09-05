package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	productService *service.ProductService
	orderService   *service.OrderService
	redisClient    *redis.Client
}

func NewHandler(productService *service.ProductService, orderService *service.OrderService, redisClient *redis.Client) *Handler {
	return &Handler{
		productService: productService,
		orderService:   orderService,
		redisClient:    redisClient,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.1.0-production",
	})
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	category := q.Get("category")

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	filter := domain.ProductFilter{
		Category: category,
		Limit:    limit,
		Offset:   offset,
	}

	products, err := h.productService.ListProducts(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	// Browser cache headers for client-side speed
	w.Header().Set("Cache-Control", "public, max-age=5")
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(products),
		"products": products,
	})
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
	}

	resp, err := h.orderService.CreateOrderAsync(r.Context(), req)
	if err != nil {
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
