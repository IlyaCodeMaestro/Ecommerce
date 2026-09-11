package service

import (
	"context"
	"log"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
)

// OrderTimeoutSweeper periodically scans and cancels stale orders, executing compensating restock
type OrderTimeoutSweeper struct {
	orderRepo    *postgres.OrderRepo
	orderService *OrderService
	timeout      time.Duration
	interval     time.Duration
}

func NewOrderTimeoutSweeper(
	orderRepo *postgres.OrderRepo,
	orderService *OrderService,
	timeout time.Duration,
	interval time.Duration,
) *OrderTimeoutSweeper {
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &OrderTimeoutSweeper{
		orderRepo:    orderRepo,
		orderService: orderService,
		timeout:      timeout,
		interval:     interval,
	}
}

// Start launches the background sweeping loop
func (s *OrderTimeoutSweeper) Start(ctx context.Context) {
	log.Printf("[SWEEPER] Starting OrderTimeoutSweeper (timeout: %v, check interval: %v)", s.timeout, s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[SWEEPER] OrderTimeoutSweeper stopped gracefully")
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *OrderTimeoutSweeper) sweep(ctx context.Context) {
	staleOrders, err := s.orderRepo.GetStaleOrders(ctx, s.timeout, 50)
	if err != nil {
		log.Printf("[SWEEPER] Error querying stale orders: %v", err)
		return
	}

	if len(staleOrders) == 0 {
		return
	}

	log.Printf("[SWEEPER] Found %d stale unpaid orders pending timeout", len(staleOrders))
	for _, order := range staleOrders {
		reason := "Automatic timeout: order unpaid for > 15 minutes"
		_, err := s.orderService.CancelOrder(ctx, order.ID, "", domain.RoleAdmin, reason)
		if err != nil {
			log.Printf("[SWEEPER] Failed to cancel stale order %s: %v", order.ID, err)
		} else {
			log.Printf("[SWEEPER] Successfully cancelled stale order %s and restored stock", order.ID)
		}
	}
}

