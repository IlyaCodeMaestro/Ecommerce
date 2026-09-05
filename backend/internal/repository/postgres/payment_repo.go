package postgres

import (
	"context"
	"errors"
	"fmt"

	"ecommerce-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type PaymentRepo struct {
	db *DB
}

func NewPaymentRepo(db *DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, p domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, amount, currency, status, idempotency_key, provider, signature, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`
	_, err := r.db.Pool.Exec(ctx, query,
		p.ID, p.OrderID, p.Amount, p.Currency, p.Status, p.IdempotencyKey, p.Provider, p.Signature,
	)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}
	return nil
}

func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, idempotency_key, provider, COALESCE(signature, ''), created_at, updated_at
		FROM payments
		WHERE id = $1
	`
	var p domain.Payment
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Status, &p.IdempotencyKey, &p.Provider, &p.Signature, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query payment by id: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepo) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus, signature string) error {
	query := `
		UPDATE payments
		SET status = $2, signature = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, id, status, signature)
	return err
}

func (r *PaymentRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id string, status domain.PaymentStatus, signature string) error {
	query := `
		UPDATE payments
		SET status = $2, signature = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, id, status, signature)
	return err
}
