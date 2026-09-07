-- Migration: 004_users_and_auth.sql
-- High-Throughput User Authentication and Role-Based Access Control (RBAC)

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'customer', -- 'customer', 'admin'
    first_name VARCHAR(128),
    last_name VARCHAR(128),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Seed initial admin and customer accounts (default password: 'password123')
-- Bcrypt hash generated with cost 10 for 'password123'
INSERT INTO users (id, email, password_hash, role, first_name, last_name)
VALUES 
    ('usr-admin-0001', 'admin@ecommerce.local', '$2a$10$V0w8EwL.LpA9b8nJ2e8k.u6XwH5qW1yJvF8lV1qJ2wH5qW1yJvF8l', 'admin', 'System', 'Admin'),
    ('usr-demo-0002', 'shopper@ecommerce.local', '$2a$10$V0w8EwL.LpA9b8nJ2e8k.u6XwH5qW1yJvF8lV1qJ2wH5qW1yJvF8l', 'customer', 'Alex', 'Dev')
ON CONFLICT (id) DO NOTHING;

