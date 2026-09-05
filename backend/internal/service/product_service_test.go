package service

import (
	"testing"
	"time"

	"ecommerce-backend/internal/domain"
)

func TestProductFilter_Defaults(t *testing.T) {
	filter := domain.ProductFilter{
		Query:    "laptop",
		Category: "laptops",
		Limit:    20,
		Offset:   0,
	}

	if filter.Query != "laptop" {
		t.Fatalf("expected query 'laptop', got '%s'", filter.Query)
	}
	if filter.Category != "laptops" {
		t.Fatalf("expected category 'laptops', got '%s'", filter.Category)
	}
}

func TestMemoryCache_SearchCacheKey(t *testing.T) {
	cache := NewMemoryCache()

	key := "search:q:laptop:cat:laptops:min:500.00:max:2000.00:sort:price_asc:lim:20:off:0"
	expected := &domain.ProductSearchResult{
		Total: 5,
		Products: []domain.Product{
			{ID: 1, Name: "Test Laptop", Price: 999.99},
		},
		Limit:  20,
		Offset: 0,
	}

	cache.Set(key, expected, 5*time.Second)

	val, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected search result to be in L1 memory cache")
	}

	res := val.(*domain.ProductSearchResult)
	if res.Total != 5 || len(res.Products) != 1 {
		t.Fatalf("unexpected search result retrieved: %+v", res)
	}
}
