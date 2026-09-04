package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-backend/internal/queue/kafka"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/internal/service"
	transporthttp "ecommerce-backend/internal/transport/http"
)

func main() {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ecommerce?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	kafkaBroker := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "orders.created")

	log.Printf("[API] Starting high-load e-commerce backend service...")
	log.Printf("[API] Config: Port=%s, DB=%s, Redis=%s, Kafka=%s, Topic=%s", port, dbURL, redisAddr, kafkaBroker, kafkaTopic)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. PostgreSQL connection pool
	db, err := postgres.NewDB(ctx, dbURL)
	if err != nil {
		log.Fatalf("[FATAL] Postgres init failed: %v", err)
	}
	defer db.Close()
	log.Printf("[API] Connected to PostgreSQL with optimized pool")

	// 2. Redis connection pool
	redisClient, err := redis.NewClient(redisAddr)
	if err != nil {
		log.Fatalf("[FATAL] Redis init failed: %v", err)
	}
	defer redisClient.Close()
	log.Printf("[API] Connected to Redis and loaded stock reservation scripts")

	// 3. Kafka async batch producer
	kafkaProducer := kafka.NewProducer(kafkaBroker, kafkaTopic)
	defer kafkaProducer.Close()
	log.Printf("[API] Initialized high-throughput Kafka producer")

	// 4. Repositories and Services
	productRepo := postgres.NewProductRepo(db)
	orderRepo := postgres.NewOrderRepo(db)

	productService := service.NewProductService(productRepo, redisClient)
	orderService := service.NewOrderService(orderRepo, productRepo, redisClient, kafkaProducer, productService)

	// 5. HTTP Handler & Router
	handler := transporthttp.NewHandler(productService, orderService)
	router := transporthttp.NewRouter(handler)

	// 6. Tuned HTTP Server for maximum throughput
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[API] Server listening on http://0.0.0.0:%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("[API] Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[API] Server forced to shutdown: %v", err)
	}
	log.Println("[API] Server stopped.")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
