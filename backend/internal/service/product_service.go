package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/pkg/metrics"
	"golang.org/x/sync/singleflight"
)

type ProductService struct {
	repo        *postgres.ProductRepo
	redisClient *redis.Client
	l1Cache     *MemoryCache
	sfg         singleflight.Group
}

func NewProductService(repo *postgres.ProductRepo, redisClient *redis.Client) *ProductService {
	return &ProductService{
		repo:        repo,
		redisClient: redisClient,
		l1Cache:     NewMemoryCache(),
	}
}

func (s *ProductService) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	key := fmt.Sprintf("product:%d", id)

	// 1. Check L1 Memory Cache (< 0.05ms)
	if val, ok := s.l1Cache.Get(key); ok {
		metrics.CacheOperationsTotal.WithLabelValues("l1", "hit").Inc()
		return val.(*domain.Product), nil
	}
	metrics.CacheOperationsTotal.WithLabelValues("l1", "miss").Inc()

	// 2. Check L2 Redis Cache (< 1ms)
	cachedJSON, err := s.redisClient.Get(ctx, key)
	if err == nil && cachedJSON != "" {
		var p domain.Product
		if err := json.Unmarshal([]byte(cachedJSON), &p); err == nil {
			metrics.CacheOperationsTotal.WithLabelValues("l2", "hit").Inc()
			// Populate L1 cache for 5 seconds to absorb spikes
			s.l1Cache.Set(key, &p, 5*time.Second)
			return &p, nil
		}
	}
	metrics.CacheOperationsTotal.WithLabelValues("l2", "miss").Inc()

	// 3. Singleflight protection: exactly 1 in-flight DB query among concurrent requests
	v, err, _ := s.sfg.Do(key, func() (interface{}, error) {
		product, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, nil
		}

		// Store in Redis (TTL 10 min)
		data, _ := json.Marshal(product)
		_ = s.redisClient.Set(ctx, key, string(data), 10*time.Minute)

		// Seed Redis stock if not present
		_ = s.redisClient.SetStock(ctx, product.ID, product.StockQuantity)

		return product, nil
	})

	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	product := v.(*domain.Product)
	s.l1Cache.Set(key, product, 5*time.Second)
	return product, nil
}

// SearchProducts handles FTS search with L1/L2 multi-tier caching and Singleflight stampede guard
func (s *ProductService) SearchProducts(ctx context.Context, filter domain.ProductFilter) (*domain.ProductSearchResult, error) {
	var minVal, maxVal float64
	if filter.MinPrice != nil {
		minVal = *filter.MinPrice
	}
	if filter.MaxPrice != nil {
		maxVal = *filter.MaxPrice
	}

	cacheKey := fmt.Sprintf("search:q:%s:cat:%s:min:%.2f:max:%.2f:sort:%s:lim:%d:off:%d",
		filter.Query, filter.Category, minVal, maxVal, filter.SortBy, filter.Limit, filter.Offset,
	)

	// 1. L1 Memory Cache (< 0.05ms)
	if val, ok := s.l1Cache.Get(cacheKey); ok {
		metrics.CacheOperationsTotal.WithLabelValues("l1", "hit").Inc()
		return val.(*domain.ProductSearchResult), nil
	}
	metrics.CacheOperationsTotal.WithLabelValues("l1", "miss").Inc()

	// 2. L2 Redis Cache (< 1ms)
	cachedJSON, err := s.redisClient.Get(ctx, cacheKey)
	if err == nil && cachedJSON != "" {
		var res domain.ProductSearchResult
		if err := json.Unmarshal([]byte(cachedJSON), &res); err == nil {
			metrics.CacheOperationsTotal.WithLabelValues("l2", "hit").Inc()
			s.l1Cache.Set(cacheKey, &res, 5*time.Second)
			return &res, nil
		}
	}
	metrics.CacheOperationsTotal.WithLabelValues("l2", "miss").Inc()

	// 3. Singleflight protection: collapse identical concurrent search bursts into 1 DB query
	v, err, _ := s.sfg.Do(cacheKey, func() (interface{}, error) {
		res, err := s.repo.Search(ctx, filter)
		if err != nil {
			return nil, err
		}

		data, _ := json.Marshal(res)
		// Cache search results in Redis for 2 minutes
		_ = s.redisClient.Set(ctx, cacheKey, string(data), 2*time.Minute)

		return res, nil
	})

	if err != nil {
		return nil, err
	}

	searchResult := v.(*domain.ProductSearchResult)
	s.l1Cache.Set(cacheKey, searchResult, 5*time.Second)
	return searchResult, nil
}

func (s *ProductService) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, error) {
	res, err := s.SearchProducts(ctx, filter)
	if err != nil {
		return nil, err
	}
	return res.Products, nil
}

func (s *ProductService) GetCategories(ctx context.Context) ([]string, error) {
	const key = "categories:all"
	if val, ok := s.l1Cache.Get(key); ok {
		return val.([]string), nil
	}

	categories, err := s.repo.GetCategories(ctx)
	if err != nil {
		return nil, err
	}
	s.l1Cache.Set(key, categories, 1*time.Hour)
	return categories, nil
}
