package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/queue/kafka"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"github.com/google/uuid"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidRequest    = errors.New("invalid order request: at least one item required")
	ErrOrderInProgress   = errors.New("order is currently being processed")
	ErrInvalidSignature  = errors.New("invalid webhook signature")
	ErrOrderNotFound     = errors.New("order not found")
)

type OrderService struct {
	orderRepo      *postgres.OrderRepo
	productRepo    *postgres.ProductRepo
	outboxRepo     *postgres.OutboxRepo
	paymentRepo    *postgres.PaymentRepo
	redisClient    *redis.Client
	kafkaProducer  *kafka.Producer
	productService *ProductService
	webhookSecret  string
}

func NewOrderService(
	orderRepo *postgres.OrderRepo,
	productRepo *postgres.ProductRepo,
	outboxRepo *postgres.OutboxRepo,
	paymentRepo *postgres.PaymentRepo,
	redisClient *redis.Client,
	kafkaProducer *kafka.Producer,
	productService *ProductService,
	webhookSecret string,
) *OrderService {
	if webhookSecret == "" {
		webhookSecret = "whsec_highload_ecommerce_2026_super_secret"
	}
	return &OrderService{
		orderRepo:      orderRepo,
		productRepo:    productRepo,
		outboxRepo:     outboxRepo,
		paymentRepo:    paymentRepo,
		redisClient:    redisClient,
		kafkaProducer:  kafkaProducer,
		productService: productService,
		webhookSecret:  webhookSecret,
	}
}

func (s *OrderService) CreateOrderAsync(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrInvalidRequest
	}

	// 1. Check Idempotency Key in Redis to prevent double-order & double-charging
	if req.IdempotencyKey != "" && s.redisClient != nil {
		acquired, state, cachedResp, err := s.redisClient.AcquireIdempotency(ctx, req.IdempotencyKey, 60*time.Second)
		if err != nil {
			return nil, fmt.Errorf("idempotency check failed: %w", err)
		}
		if !acquired {
			if state == "IN_PROGRESS" {
				return nil, ErrOrderInProgress
			}
			if state == "COMPLETED" && cachedResp != "" {
				var cachedObj domain.CreateOrderResponse
				if err := json.Unmarshal([]byte(cachedResp), &cachedObj); err == nil {
					return &cachedObj, nil
				}
			}
		}
	}

	orderID := uuid.New().String()
	now := time.Now().UTC()
	var totalAmount float64
	orderItems := make([]domain.OrderItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		if itemReq.Quantity <= 0 {
			s.releaseIdempotency(ctx, req.IdempotencyKey)
			return nil, errors.New("quantity must be positive")
		}

		// Fetch product price & details
		product, err := s.productService.GetProductByID(ctx, itemReq.ProductID)
		if err != nil {
			s.releaseIdempotency(ctx, req.IdempotencyKey)
			return nil, fmt.Errorf("failed to fetch product: %w", err)
		}
		if product == nil {
			s.releaseIdempotency(ctx, req.IdempotencyKey)
			return nil, ErrProductNotFound
		}

		// Atomic stock reservation via Redis Lua script
		if s.redisClient != nil {
			res, err := s.redisClient.ReserveStock(ctx, itemReq.ProductID, itemReq.Quantity)
			if err != nil {
				s.releaseIdempotency(ctx, req.IdempotencyKey)
				return nil, fmt.Errorf("redis stock reservation failed: %w", err)
			}

			if res == -1 {
				_ = s.redisClient.SetStock(ctx, product.ID, product.StockQuantity)
				res, err = s.redisClient.ReserveStock(ctx, itemReq.ProductID, itemReq.Quantity)
				if err != nil || res != 1 {
					s.releaseIdempotency(ctx, req.IdempotencyKey)
					return nil, ErrInsufficientStock
				}
			} else if res == 0 {
				s.releaseIdempotency(ctx, req.IdempotencyKey)
				return nil, ErrInsufficientStock
			}
		}

		itemTotal := product.Price * float64(itemReq.Quantity)
		totalAmount += itemTotal

		orderItems = append(orderItems, domain.OrderItem{
			OrderID:   orderID,
			ProductID: product.ID,
			Quantity:  itemReq.Quantity,
			Price:     product.Price,
		})
	}

	// 2. Prepare OrderPlacedEvent
	event := domain.OrderPlacedEvent{
		OrderID:     orderID,
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Items:       orderItems,
		CreatedAt:   now,
	}

	eventPayload, _ := json.Marshal(event)

	// 3. Atomically save Order + Transactional Outbox Event in PostgreSQL
	order := domain.Order{
		ID:          orderID,
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Status:      domain.OrderStatusAccepted,
		Items:       orderItems,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	outboxEvent := domain.OutboxEvent{
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     "OrderPlaced",
		Payload:       eventPayload,
		Status:        "PENDING",
		CreatedAt:     now,
	}

	if err := s.orderRepo.SaveSingleWithOutbox(ctx, order, outboxEvent); err != nil {
		s.releaseIdempotency(ctx, req.IdempotencyKey)
		return nil, fmt.Errorf("failed to persist order and outbox event: %w", err)
	}

	// 4. Also publish to Kafka for fast-path processing
	if s.kafkaProducer != nil {
		_ = s.kafkaProducer.PublishOrder(ctx, event)
	}

	resp := &domain.CreateOrderResponse{
		OrderID:   orderID,
		Status:    domain.OrderStatusAccepted,
		Message:   "Order accepted and queued for processing (Outbox & Idempotency active)",
		CreatedAt: now,
	}

	// 5. Cache completed response in Redis for 24h to fulfill subsequent duplicate requests
	if req.IdempotencyKey != "" && s.redisClient != nil {
		respJSON, _ := json.Marshal(resp)
		_ = s.redisClient.SaveIdempotency(ctx, req.IdempotencyKey, string(respJSON), 24*time.Hour)
	}

	return resp, nil
}

func (s *OrderService) ProcessPaymentWebhook(ctx context.Context, signature string, rawBody []byte, payload domain.PaymentWebhookPayload) error {
	// 1. Cryptographic HMAC-SHA256 signature verification
	if !s.verifySignature(rawBody, signature) {
		return ErrInvalidSignature
	}

	if payload.OrderID == "" {
		return errors.New("missing order_id in webhook payload")
	}

	// 2. Fetch order to verify existence
	order, err := s.orderRepo.GetByID(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return ErrOrderNotFound
	}

	// 3. Idempotent webhook check: if already completed, return nil without duplicate processing
	if order.Status == domain.OrderStatusCompleted || order.Status == domain.OrderStatusPaid {
		return nil
	}

	// 4. Update order status in PostgreSQL
	if err := s.orderRepo.UpdateStatus(ctx, payload.OrderID, domain.OrderStatusCompleted); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// 5. Persist Payment audit record
	if s.paymentRepo != nil {
		payment := domain.Payment{
			ID:        payload.PaymentID,
			OrderID:   payload.OrderID,
			Amount:    payload.Amount,
			Currency:  payload.Currency,
			Status:    domain.PaymentStatusSucceeded,
			Provider:  "stripe",
			Signature: signature,
		}
		_ = s.paymentRepo.Create(ctx, payment)
	}

	// 6. Broadcast real-time SSE update via Redis Pub/Sub to frontend
	if s.redisClient != nil {
		channelName := fmt.Sprintf("order:%s:status", payload.OrderID)
		sseMsg := fmt.Sprintf(`{"order_id":"%s","status":"COMPLETED","step":3,"message":"Payment verified via HMAC-SHA256 • Order Completed","timestamp":"%s"}`,
			payload.OrderID, time.Now().UTC().Format(time.RFC3339),
		)
		_ = s.redisClient.Publish(ctx, channelName, sseMsg)
	}

	return nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *OrderService) releaseIdempotency(ctx context.Context, key string) {
	if key != "" && s.redisClient != nil {
		_ = s.redisClient.ReleaseIdempotency(ctx, key)
	}
}

func (s *OrderService) verifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

func (s *OrderService) GenerateWebhookSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
