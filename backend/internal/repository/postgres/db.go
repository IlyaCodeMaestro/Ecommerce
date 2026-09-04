package postgres

import (
	"context"
	"fmt"
	"time"

	"ecommerce-backend/pkg/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database url: %w", err)
	}

	// High concurrency tuning for 10,000 RPS
	config.MaxConns = 250
	config.MinConns = 25
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetime = 30 * time.Minute
	config.HealthCheckPeriod = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	db := &DB{Pool: pool}
	db.startPoolMetrics(ctx)

	return db, nil
}

func (db *DB) startPoolMetrics(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stat := db.Pool.Stat()
				metrics.DBPoolAcquiredConns.Set(float64(stat.AcquiredConns()))
				metrics.DBPoolIdleConns.Set(float64(stat.IdleConns()))
			}
		}
	}()
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
