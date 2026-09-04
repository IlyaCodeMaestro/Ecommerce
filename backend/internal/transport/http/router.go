package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(handler *Handler) *chi.Mux {
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

		r.Post("/orders", handler.CreateOrder)
		r.Get("/orders/{id}", handler.GetOrderByID)
	})

	return r
}
