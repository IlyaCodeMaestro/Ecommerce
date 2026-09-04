package http

import (
	"ecommerce-backend/internal/repository/redis"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(handler *Handler, redisClient *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	// High-throughput middleware stack
	r.Use(CORSMiddleware)
	r.Use(MetricsMiddleware)
	r.Use(middleware.Recoverer)

	// Metrics & Probe endpoints
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/healthz", handler.HealthCheck)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/categories", handler.GetCategories)
		r.Get("/products", handler.ListProducts)
		r.Get("/products/{id}", handler.GetProductByID)

		// Orders endpoints with rate limiting on order creation
		r.Group(func(orderRouter chi.Router) {
			orderRouter.Use(RateLimitMiddleware(redisClient, 40, 60)) // 40 orders/min per IP
			orderRouter.Post("/orders", handler.CreateOrder)
		})

		r.Get("/orders/{id}", handler.GetOrderByID)
		r.Get("/orders/{id}/stream", handler.StreamOrderStatus) // Real-time SSE stream
	})

	return r
}
