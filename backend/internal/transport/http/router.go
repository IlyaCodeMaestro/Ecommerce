package http

import (
	"ecommerce-backend/internal/domain"
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
	r.Get("/readyz", handler.ReadinessCheck)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication endpoints with brute force protection
		r.Route("/auth", func(authRouter chi.Router) {
			authRouter.Use(RateLimitMiddleware(redisClient, 30, 60)) // 30 req/min per IP
			authRouter.Post("/register", handler.Register)
			authRouter.Post("/login", handler.Login)
			authRouter.Post("/refresh", handler.RefreshToken)
			authRouter.Post("/logout", handler.Logout)

			// Protected user profile
			authRouter.Group(func(protected chi.Router) {
				protected.Use(AuthMiddleware(handler.authService))
				protected.Get("/me", handler.GetMe)
			})
		})

		r.Get("/categories", handler.GetCategories)
		r.Get("/products", handler.ListProducts)
		r.Get("/products/{id}", handler.GetProductByID)

		// Orders endpoints with rate limiting on order creation
		// Cart endpoints (Supports both Guest Sessions and Authenticated Users)
		r.Route("/cart", func(cartRouter chi.Router) {
			cartRouter.Use(OptionalAuthMiddleware(handler.authService))
			cartRouter.Get("/", handler.GetCart)
			cartRouter.Post("/items", handler.AddCartItem)
			cartRouter.Put("/items/{id}", handler.UpdateCartItem)
			cartRouter.Delete("/items/{id}", handler.RemoveCartItem)
			cartRouter.Delete("/", handler.ClearCart)
			cartRouter.Post("/merge", handler.MergeCart)
		})

		// Orders endpoints with rate limiting & optional/authenticated JWT context
		r.Group(func(orderRouter chi.Router) {
			orderRouter.Use(RateLimitMiddleware(redisClient, 40, 60)) // 40 orders/min per IP
			orderRouter.Use(OptionalAuthMiddleware(handler.authService))
			orderRouter.Post("/orders", handler.CreateOrder)
			orderRouter.Get("/orders", handler.ListUserOrders)
			orderRouter.Put("/orders/{id}/cancel", handler.CancelOrder)
		})

		r.Get("/orders/{id}", handler.GetOrderByID)
		r.Get("/orders/{id}/stream", handler.StreamOrderStatus) // Real-time SSE stream

		// Payments & Webhook endpoints
		r.Post("/payments/webhook", handler.PaymentWebhook)
		r.Post("/payments/simulate", handler.SimulatePayment)

		// Protected Admin routes (RBAC demonstration)
		r.Group(func(adminRouter chi.Router) {
			adminRouter.Use(AuthMiddleware(handler.authService))
			adminRouter.Use(RequireRole(domain.RoleAdmin))
			adminRouter.Get("/admin/stats", handler.GetAdminStats)
			adminRouter.Get("/admin/orders", handler.AdminListOrders)
			adminRouter.Put("/admin/orders/{id}/status", handler.AdminUpdateOrderStatus)
		})

		// Internal Alertmanager Webhook receivers
		r.Post("/internal/alerts", handler.AlertWebhook)
		r.Post("/internal/alerts/critical", handler.AlertWebhook)
	})

	return r
}
