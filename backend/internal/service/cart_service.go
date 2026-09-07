package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
)

var (
	ErrInvalidQuantity = errors.New("quantity must be greater than 0")
)

const CartTTL = 30 * 24 * time.Hour // 30 days cart persistence

type CartService struct {
	redisClient *redis.Client
	productRepo *postgres.ProductRepo
}

func NewCartService(redisClient *redis.Client, productRepo *postgres.ProductRepo) *CartService {
	return &CartService{
		redisClient: redisClient,
		productRepo: productRepo,
	}
}

// GetCart retrieves the active cart from Redis or returns a fresh initialized cart
func (s *CartService) GetCart(ctx context.Context, cartKey string) (*domain.Cart, error) {
	if s.redisClient == nil {
		return &domain.Cart{
			CartKey: cartKey,
			Items:   []domain.CartItem{},
		}, nil
	}

	cart, err := s.redisClient.GetCart(ctx, cartKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cart from redis: %w", err)
	}

	if cart == nil {
		return &domain.Cart{
			CartKey: cartKey,
			Items:   []domain.CartItem{},
		}, nil
	}

	return cart, nil
}

// AddItem adds a product or increments quantity in the Redis cart
func (s *CartService) AddItem(ctx context.Context, cartKey string, req domain.AddToCartRequest) (*domain.Cart, error) {
	if req.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	cart, err := s.GetCart(ctx, cartKey)
	if err != nil {
		return nil, err
	}

	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == req.ProductID {
			newQty := cart.Items[i].Quantity + req.Quantity
			if product.StockQuantity < newQty {
				return nil, ErrInsufficientStock
			}
			cart.Items[i].Quantity = newQty
			cart.Items[i].Price = product.Price // Sync fresh price
			found = true
			break
		}
	}

	if !found {
		if product.StockQuantity < req.Quantity {
			return nil, ErrInsufficientStock
		}
		cart.Items = append(cart.Items, domain.CartItem{
			ProductID: product.ID,
			SKU:       product.SKU,
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  req.Quantity,
		})
	}

	cart.Recalculate()

	if s.redisClient != nil {
		if err := s.redisClient.SaveCart(ctx, cart, CartTTL); err != nil {
			return nil, fmt.Errorf("failed to persist cart in redis: %w", err)
		}
	}

	return cart, nil
}

// UpdateItem updates the exact quantity of a product in the cart
func (s *CartService) UpdateItem(ctx context.Context, cartKey string, productID int64, quantity int) (*domain.Cart, error) {
	if quantity <= 0 {
		return s.RemoveItem(ctx, cartKey, productID)
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	if product.StockQuantity < quantity {
		return nil, ErrInsufficientStock
	}

	cart, err := s.GetCart(ctx, cartKey)
	if err != nil {
		return nil, err
	}

	found := false
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			cart.Items[i].Quantity = quantity
			cart.Items[i].Price = product.Price
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, domain.CartItem{
			ProductID: product.ID,
			SKU:       product.SKU,
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  quantity,
		})
	}

	cart.Recalculate()

	if s.redisClient != nil {
		if err := s.redisClient.SaveCart(ctx, cart, CartTTL); err != nil {
			return nil, fmt.Errorf("failed to update cart in redis: %w", err)
		}
	}

	return cart, nil
}

// RemoveItem removes a line item from the cart
func (s *CartService) RemoveItem(ctx context.Context, cartKey string, productID int64) (*domain.Cart, error) {
	cart, err := s.GetCart(ctx, cartKey)
	if err != nil {
		return nil, err
	}

	newItems := make([]domain.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item.ProductID != productID {
			newItems = append(newItems, item)
		}
	}
	cart.Items = newItems
	cart.Recalculate()

	if s.redisClient != nil {
		if err := s.redisClient.SaveCart(ctx, cart, CartTTL); err != nil {
			return nil, fmt.Errorf("failed to save cart after removal: %w", err)
		}
	}

	return cart, nil
}

// ClearCart removes all items and deletes cart key from Redis
func (s *CartService) ClearCart(ctx context.Context, cartKey string) error {
	if s.redisClient != nil {
		return s.redisClient.DeleteCart(ctx, cartKey)
	}
	return nil
}

// MergeCart merges guest cart items into a user's authenticated cart
func (s *CartService) MergeCart(ctx context.Context, userCartKey string, guestItems []domain.AddToCartRequest) (*domain.Cart, error) {
	cart, err := s.GetCart(ctx, userCartKey)
	if err != nil {
		return nil, err
	}

	for _, gItem := range guestItems {
		if gItem.Quantity <= 0 {
			continue
		}

		product, err := s.productRepo.GetByID(ctx, gItem.ProductID)
		if err != nil || product == nil {
			continue // Skip unavailable items during merge
		}

		found := false
		for i := range cart.Items {
			if cart.Items[i].ProductID == gItem.ProductID {
				mergedQty := cart.Items[i].Quantity + gItem.Quantity
				if mergedQty > product.StockQuantity {
					mergedQty = product.StockQuantity
				}
				cart.Items[i].Quantity = mergedQty
				cart.Items[i].Price = product.Price
				found = true
				break
			}
		}

		if !found {
			qty := gItem.Quantity
			if qty > product.StockQuantity {
				qty = product.StockQuantity
			}
			if qty > 0 {
				cart.Items = append(cart.Items, domain.CartItem{
					ProductID: product.ID,
					SKU:       product.SKU,
					Name:      product.Name,
					Price:     product.Price,
					Quantity:  qty,
				})
			}
		}
	}

	cart.Recalculate()

	if s.redisClient != nil {
		if err := s.redisClient.SaveCart(ctx, cart, CartTTL); err != nil {
			return nil, fmt.Errorf("failed to persist merged cart: %w", err)
		}
	}

	return cart, nil
}

