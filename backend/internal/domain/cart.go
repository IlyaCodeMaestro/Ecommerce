package domain

import "time"

type CartItem struct {
	ProductID int64   `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Subtotal  float64 `json:"subtotal"`
}

type Cart struct {
	CartKey       string     `json:"cart_key"`
	Items         []CartItem `json:"items"`
	TotalItems    int        `json:"total_items"`
	TotalAmount   float64    `json:"total_amount"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (c *Cart) Recalculate() {
	totalItems := 0
	totalAmount := 0.0

	for i := range c.Items {
		c.Items[i].Subtotal = c.Items[i].Price * float64(c.Items[i].Quantity)
		totalItems += c.Items[i].Quantity
		totalAmount += c.Items[i].Subtotal
	}

	c.TotalItems = totalItems
	c.TotalAmount = totalAmount
	c.UpdatedAt = time.Now().UTC()
}

type AddToCartRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

type MergeCartRequest struct {
	Items []AddToCartRequest `json:"items"`
}

