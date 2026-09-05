package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"ecommerce-backend/internal/queue/kafka"
	"ecommerce-backend/internal/repository/postgres"
)

type OutboxRelay struct {
	outboxRepo   *postgres.OutboxRepo
	producer     *kafka.Producer
	pollInterval time.Duration
	batchSize    int
}

func NewOutboxRelay(
	outboxRepo *postgres.OutboxRepo,
	producer *kafka.Producer,
	pollInterval time.Duration,
	batchSize int,
) *OutboxRelay {
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	return &OutboxRelay{
		outboxRepo:   outboxRepo,
		producer:     producer,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

// Start runs the background polling relay worker. It delivers outbox events
// to Kafka with guaranteed At-Least-Once semantics.
func (r *OutboxRelay) Start(ctx context.Context) {
	log.Printf("[OUTBOX RELAY] Background relay service started (interval=%v, batch=%d)", r.pollInterval, r.batchSize)
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[OUTBOX RELAY] Stopped outbox relay worker.")
			return
		case <-ticker.C:
			count, err := r.ProcessBatch(ctx)
			if err != nil {
				log.Printf("[OUTBOX RELAY ERROR] Batch processing failed: %v", err)
			} else if count > 0 {
				log.Printf("[OUTBOX RELAY] Relayed %d events to Kafka successfully", count)
			}
		}
	}
}

// ProcessBatch locks, fetches, publishes and commits a batch of pending events.
func (r *OutboxRelay) ProcessBatch(ctx context.Context) (int, error) {
	tx, err := r.outboxRepo.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin outbox tx: %w", err)
	}
	defer tx.Rollback(ctx)

	events, err := r.outboxRepo.FetchPendingTx(ctx, tx, r.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pending events: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	var publishedIDs []string

	for _, event := range events {
		// Publish event payload to Kafka topic
		if err := r.producer.PublishRaw(ctx, event.AggregateID, event.Payload); err != nil {
			_ = r.outboxRepo.MarkFailedTx(ctx, tx, event.ID, err.Error())
			continue
		}
		publishedIDs = append(publishedIDs, event.ID)
	}

	if len(publishedIDs) > 0 {
		if err := r.outboxRepo.MarkPublishedTx(ctx, tx, publishedIDs); err != nil {
			return 0, fmt.Errorf("failed to mark events published: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit outbox relay tx: %w", err)
	}

	return len(publishedIDs), nil
}
