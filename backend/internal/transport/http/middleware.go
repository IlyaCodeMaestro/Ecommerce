package http

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/pkg/metrics"
	"github.com/go-chi/chi/v5/middleware"
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())
		path := r.URL.Path

		// Group parameterized URLs to prevent high cardinality in Prometheus
		if len(path) > 17 && path[:17] == "/api/v1/products/" {
			path = "/api/v1/products/:id"
		} else if len(path) > 15 && path[:15] == "/api/v1/orders/" {
			if strings.HasSuffix(path, "/stream") {
				path = "/api/v1/orders/:id/stream"
			} else {
				path = "/api/v1/orders/:id"
			}
		}

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware enforces token-bucket rate limits per client IP
func RateLimitMiddleware(redisClient *redis.Client, limit int, windowSeconds int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redisClient == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			key := fmt.Sprintf("ratelimit:%s:%s", r.URL.Path, ip)

			allowed, err := redisClient.AllowRequest(r.Context(), key, limit, windowSeconds)
			if err != nil || !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(windowSeconds))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Rate limit exceeded. Too many requests. Please slow down."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
