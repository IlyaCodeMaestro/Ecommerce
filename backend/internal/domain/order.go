package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusAccepted   OrderStatus = "ACCEPTED"
	OrderStatusPaid       OrderStatus = "PAID"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusDelivered  OrderStatus = "DELIVERED"
	OrderStatusCompleted  OrderStatus = "COMPLETED" // Compatible alias with DELIVERED
	OrderStatusCancelled  OrderStatus = "CANCELLED"
	OrderStatusRefunded   OrderStatus = "REFUNDED"
	OrderStatusFailed     OrderStatus = "FAILED"
)

// CanTransitionTo validates state progression according to e-commerce FSM rules
func (current OrderStatus) CanTransitionTo(next OrderStatus) bool {
	if current == next {
		return true // Idempotent no-op
	}

	switch current {
	case OrderStatusAccepted, OrderStatusPending:
		return next == OrderStatusPaid || next == OrderStatusCancelled || next == OrderStatusFailed
	case OrderStatusPaid:
		return next == OrderStatusProcessing || next == OrderStatusCancelled || next == OrderStatusRefunded
	case OrderStatusProcessing:
		return next == OrderStatusShipped || next == OrderStatusCancelled || next == OrderStatusRefunded
	case OrderStatusShipped:
		return next == OrderStatusDelivered || next == OrderStatusCompleted || next == OrderStatusRefunded
	case OrderStatusDelivered, OrderStatusCompleted:
		return next == OrderStatusRefunded
	case OrderStatusCancelled, OrderStatusRefunded, OrderStatusFailed:
		return false
	default:
		return false
	}
}

type OrderItem struct {
	ID        int64   `json:"id,omitempty"`
	OrderID   string  `json:"order_id,omitempty"`
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type CreateOrderRequest struct {
	UserID         string             `json:"user_id"`
	Items          []OrderItemRequest `json:"items"`
	IdempotencyKey string             `json:"idempotency_key,omitempty"`
}

type OrderItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type CreateOrderResponse struct {
	OrderID   string      `json:"order_id"`
	Status    OrderStatus `json:"status"`
	Message   string      `json:"message"`
	CreatedAt time.Time   `json:"created_at"`
}

// OrderPlacedEvent is serialized to Kafka for async persistence
type OrderPlacedEvent struct {
	OrderID     string      `json:"order_id"`
	UserID      string      `json:"user_id"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
}

// OutboxEvent represents an atomic transaction event for Transactional Outbox Pattern
type OutboxEvent struct {
	ID            string     `json:"id"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	EventType     string     `json:"event_type"`
	Payload       []byte     `json:"payload"`
	Status        string     `json:"status"`
	RetryCount    int        `json:"retry_count"`
	LastError     string     `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusSucceeded PaymentStatus = "SUCCEEDED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
)

type Payment struct {
	ID             string        `json:"id"`
	OrderID        string        `json:"order_id"`
	Amount         float64       `json:"amount"`
	Currency       string        `json:"currency"`
	Status         PaymentStatus `json:"status"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
	Provider       string        `json:"provider"`
	Signature      string        `json:"signature,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type PaymentWebhookPayload struct {
	Event     string  `json:"event"`
	PaymentID string  `json:"payment_id"`
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason,omitempty"`
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status"`
	Reason string      `json:"reason,omitempty"`
}

type OrderListResponse struct {
	Orders []Order `json:"orders"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}
