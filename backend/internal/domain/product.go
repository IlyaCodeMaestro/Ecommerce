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
	Query    string   `json:"query"`     // Full-Text Search term
	Category string   `json:"category"`  // Category filter
	MinPrice *float64 `json:"min_price"` // Minimum price bound
	MaxPrice *float64 `json:"max_price"` // Maximum price bound
	SortBy   string   `json:"sort_by"`   // "relevance", "price_asc", "price_desc", "newest"
	Limit    int      `json:"limit"`     // Pagination limit
	Offset   int      `json:"offset"`    // Pagination offset
}

type ProductSearchResult struct {
	Total    int64     `json:"total"`
	Products []Product `json:"products"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}
