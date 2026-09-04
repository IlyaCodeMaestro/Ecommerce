package domain

import "time"

type Product struct {
	ID            int64     `json:"id"`
	SKU           string    `json:"sku"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	Category      string    `json:"category"`
	StockQuantity int       `json:"stock_quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProductFilter struct {
	Category string
	Limit    int
	Offset   int
}
