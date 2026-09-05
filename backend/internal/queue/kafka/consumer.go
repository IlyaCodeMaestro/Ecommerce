package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/repository/postgres"
	"ecommerce-backend/internal/repository/redis"
	"ecommerce-backend/pkg/metrics"

	kafkaPkg "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader      *kafkaPkg.Reader
	orderRepo   *postgres.OrderRepo
	redisClient *redis.Client
}

func NewConsumer(broker string, topic string, groupID string, orderRepo *postgres.OrderRepo, redisClient *redis.Client) *Consumer {
	reader := kafkaPkg.NewReader(kafkaPkg.ReaderConfig{
		Brokers:        []string{broker},
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 500 * time.Millisecond,
		MaxWait:        50 * time.Millisecond,
	})

	return &Consumer{
		reader:      reader,
		orderRepo:   orderRepo,
		redisClient: redisClient,
	}
}

// StartBatchWorker consumes messages and commits in batches to Postgres
func (c *Consumer) StartBatchWorker(ctx context.Context, batchSize int, flushTimeout time.Duration) {
	fmt.Printf("[KAFKA WORKER] Starting batch worker (batchSize=%d, flushTimeout=%v)...\n", batchSize, flushTimeout)

	batch := make([]domain.OrderPlacedEvent, 0, batchSize)
	ticker := time.NewTicker(flushTimeout)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		count := len(batch)
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.orderRepo.SaveBatch(flushCtx, batch); err != nil {
			fmt.Printf("[KAFKA WORKER ERROR] Failed to batch save %d orders to Postgres: %v\n", count, err)
		} else {
			metrics.KafkaOrdersConsumedTotal.Add(float64(count))

			// Broadcast completion event to Redis Pub/Sub for real-time SSE frontend clients
			if c.redisClient != nil {
				for _, event := range batch {
					statusPayload, _ := json.Marshal(map[string]interface{}{
						"order_id":  event.OrderID,
						"status":    "COMPLETED",
						"message":   "Order successfully persisted to PostgreSQL",
						"timestamp": time.Now().UTC(),
						"step":      3,
					})
					_ = c.redisClient.Publish(flushCtx, fmt.Sprintf("order:%s:status", event.OrderID), string(statusPayload))
				}
			}
		}
		batch = batch[:0] // reset slice keeping allocated capacity
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		default:
			// Fetch next message with short timeout
			fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
			msg, err := c.reader.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				// Timeout or context cancel is normal when queue is briefly idle
				continue
			}

			var event domain.OrderPlacedEvent
			if err := json.Unmarshal(msg.Value, &event); err == nil {
				batch = append(batch, event)
			}

			// Commit offset to Kafka
			_ = c.reader.CommitMessages(ctx, msg)

			if len(batch) >= batchSize {
				flush()
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
