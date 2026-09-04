package service

import (
	"context"
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
)

type OrderService struct {
	orderRepo      *postgres.OrderRepo
	productRepo    *postgres.ProductRepo
	redisClient    *redis.Client
	kafkaProducer  *kafka.Producer
	productService *ProductService
}

func NewOrderService(
	orderRepo *postgres.OrderRepo,
	productRepo *postgres.ProductRepo,
	redisClient *redis.Client,
	kafkaProducer *kafka.Producer,
	productService *ProductService,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		productRepo:    productRepo,
		redisClient:    redisClient,
		kafkaProducer:  kafkaProducer,
		productService: productService,
	}
}

func (s *OrderService) CreateOrderAsync(ctx context.Context, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrInvalidRequest
	}

	orderID := uuid.New().String()
	now := time.Now().UTC()
	var totalAmount float64
	orderItems := make([]domain.OrderItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		if itemReq.Quantity <= 0 {
			return nil, errors.New("quantity must be positive")
		}

		// 1. Fetch product price & details
		product, err := s.productService.GetProductByID(ctx, itemReq.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product: %w", err)
		}
		if product == nil {
			return nil, ErrProductNotFound
		}

		// 2. Atomic stock reservation via Redis Lua script
		res, err := s.redisClient.ReserveStock(ctx, itemReq.ProductID, itemReq.Quantity)
		if err != nil {
			return nil, fmt.Errorf("redis stock reservation failed: %w", err)
		}

		if res == -1 {
			// Stock key wasn't in Redis yet, initialize and retry once
			_ = s.redisClient.SetStock(ctx, product.ID, product.StockQuantity)
			res, err = s.redisClient.ReserveStock(ctx, itemReq.ProductID, itemReq.Quantity)
			if err != nil || res != 1 {
				return nil, ErrInsufficientStock
			}
		} else if res == 0 {
			return nil, ErrInsufficientStock
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

	// 3. Publish OrderPlacedEvent to Kafka
	event := domain.OrderPlacedEvent{
		OrderID:     orderID,
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Items:       orderItems,
		CreatedAt:   now,
	}

	if err := s.kafkaProducer.PublishOrder(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to enqueue order: %w", err)
	}

	return &domain.CreateOrderResponse{
		OrderID:   orderID,
		Status:    domain.OrderStatusAccepted,
		Message:   "Order accepted and queued for processing",
		CreatedAt: now,
	}, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}
