package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-backend/internal/queue/kafka"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	metricsPort := getEnv("PORT", "8081")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ecommerce?sslmode=disable")
	kafkaBroker := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "orders.created")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "order-batch-persist-group")

	log.Printf("[WORKER] Starting Kafka batch persistence worker...")
	log.Printf("[WORKER] Broker=%s, Topic=%s, GroupID=%s", kafkaBroker, kafkaTopic, kafkaGroupID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. PostgreSQL connection pool
	db, err := postgres.NewDB(ctx, dbURL)
	if err != nil {
		log.Fatalf("[FATAL] Worker Postgres connection failed: %v", err)
	}
	defer db.Close()

	orderRepo := postgres.NewOrderRepo(db)

	// 2. Metrics HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: mux,
	}

	go func() {
		log.Printf("[WORKER] Metrics server listening on :%s/metrics", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WORKER WARNING] Metrics server error: %v", err)
		}
	}()

	// 3. Redis connection for Pub/Sub notifications
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisClient, err := redis.NewClient(redisAddr)
	if err != nil {
		log.Printf("[WORKER WARNING] Redis connect failed, real-time pubsub disabled: %v", err)
	} else {
		defer redisClient.Close()
	}

	// 4. Kafka consumer
	consumer := kafka.NewConsumer(kafkaBroker, kafkaTopic, kafkaGroupID, orderRepo, redisClient)
	defer consumer.Close()

	// Launch batch worker: batchSize=200, flushTimeout=100ms
	go consumer.StartBatchWorker(ctx, 200, 100*time.Millisecond)

	// Graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	<-stopChan
	log.Println("[WORKER] Shutting down worker...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = metricsServer.Shutdown(shutdownCtx)

	log.Println("[WORKER] Worker stopped successfully.")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
