package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"ecommerce-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type OutboxRepo struct {
	db *DB
}

func NewOutboxRepo(db *DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

// SaveTx persists an outbox event within an existing PostgreSQL ACID transaction
func (r *OutboxRepo) SaveTx(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID, eventType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox event payload: %w", err)
	}

	query := `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, status, created_at)
		VALUES ($1, $2, $3, $4, 'PENDING', NOW())
	`
	_, err = tx.Exec(ctx, query, aggregateType, aggregateID, eventType, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

// FetchPendingTx fetches and locks up to limit pending events using FOR UPDATE SKIP LOCKED.
// Multiple worker replicas can call this concurrently without blocking each other or deadlocking.
func (r *OutboxRepo) FetchPendingTx(ctx context.Context, tx pgx.Tx, limit int) ([]domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, status, retry_count, COALESCE(last_error, ''), created_at, processed_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		var payloadRaw []byte
		if err := rows.Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.EventType,
			&payloadRaw,
			&e.Status,
			&e.RetryCount,
			&e.LastError,
			&e.CreatedAt,
			&e.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		e.Payload = payloadRaw
		events = append(events, e)
	}

	return events, nil
}

// MarkPublishedTx marks events as PUBLISHED with current timestamp inside the transaction
func (r *OutboxRepo) MarkPublishedTx(ctx context.Context, tx pgx.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE outbox_events
		SET status = 'PUBLISHED', processed_at = NOW()
		WHERE id = ANY($1)
	`
	_, err := tx.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark outbox events as published: %w", err)
	}
	return nil
}

// MarkFailedTx increments retry count and sets last error message
func (r *OutboxRepo) MarkFailedTx(ctx context.Context, tx pgx.Tx, id string, errStr string) error {
	query := `
		UPDATE outbox_events
		SET retry_count = retry_count + 1, last_error = $2,
		    status = CASE WHEN retry_count + 1 >= 5 THEN 'FAILED' ELSE 'PENDING' END
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, id, errStr)
	return err
}

// BeginTx starts a new transaction on the DB pool
func (r *OutboxRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Pool.Begin(ctx)
}
