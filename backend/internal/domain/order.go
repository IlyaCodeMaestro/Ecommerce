package domain

import "time"

type OrderStatus string

const (
	OrderStatusAccepted   OrderStatus = "ACCEPTED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusFailed     OrderStatus = "FAILED"
)

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
	UserID string             `json:"user_id"`
	Items  []OrderItemRequest `json:"items"`
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
