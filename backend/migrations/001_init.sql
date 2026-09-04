-- Create tables
CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(12, 2) NOT NULL,
    category VARCHAR(64) NOT NULL,
    stock_quantity INT NOT NULL DEFAULT 100000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    total_amount NUMERIC(12, 2) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACCEPTED',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products(id),
    quantity INT NOT NULL,
    price NUMERIC(12, 2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

-- Seed initial catalog of 1,000 products for realistic load benchmarking
INSERT INTO products (sku, name, description, price, category, stock_quantity)
SELECT 
    'SKU-' || lpad(i::text, 5, '0'),
    CASE (i % 5)
        WHEN 0 THEN 'UltraBook Pro ' || (13 + (i % 4)) || '" M' || ((i % 3) + 1)
        WHEN 1 THEN 'CyberPhone Max ' || ((i % 5) + 12) || ' Pro'
        WHEN 2 THEN 'Studio Headphones Wireless ' || (100 + i)
        WHEN 3 THEN '4K Gaming Monitor ' || (24 + (i % 10)) || '" 144Hz'
        ELSE 'Mechanical Keyboard RGB ' || (i + 1)
    END,
    'High-performance item designed for extreme reliability, testing, and production workloads. Series #' || i,
    (29.99 + (i % 950) * 1.5)::numeric(12,2),
    CASE (i % 5)
        WHEN 0 THEN 'laptops'
        WHEN 1 THEN 'smartphones'
        WHEN 2 THEN 'audio'
        WHEN 3 THEN 'monitors'
        ELSE 'accessories'
    END,
    1000000 -- generous stock for load testing
FROM generate_series(1, 1000) AS i
ON CONFLICT (sku) DO NOTHING;
