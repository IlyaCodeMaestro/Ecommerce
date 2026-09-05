-- Migration: 003_outbox_and_payments.sql
-- Transactional Outbox Pattern & Idempotent Payments for High-Load Architecture

-- 1. Transactional Outbox Events Table
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,              -- e.g. 'order', 'payment'
    aggregate_id VARCHAR(128) NOT NULL,               -- e.g. order_id
    event_type VARCHAR(64) NOT NULL,                  -- e.g. 'OrderPlaced', 'PaymentReceived', 'OrderCompleted'
    payload JSONB NOT NULL,                           -- serialized event payload
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',    -- 'PENDING', 'PUBLISHED', 'FAILED'
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Partial Index: Ultra-fast index covering ONLY unprocessed events (< 0.1ms lookups on millions of rows)
CREATE INDEX IF NOT EXISTS idx_outbox_pending 
ON outbox_events (created_at ASC) 
WHERE status = 'PENDING';

-- 2. Payments Table
CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(128) PRIMARY KEY,
    order_id VARCHAR(128) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    status VARCHAR(32) NOT NULL,                      -- 'PENDING', 'SUCCEEDED', 'FAILED'
    idempotency_key VARCHAR(128) UNIQUE,
    provider VARCHAR(64) NOT NULL DEFAULT 'stripe',
    signature VARCHAR(256),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency ON payments (idempotency_key);
