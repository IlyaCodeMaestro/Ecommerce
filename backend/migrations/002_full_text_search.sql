-- Migration 002: Full-Text Search with GIN Indexes and Advanced Filtering Indexes

-- 1. Create GIN index for high-speed full-text search across name and description
-- GIN (Generalized Inverted Index) maps lexemes to row pointers, turning O(N) table scans into O(log N) index lookups.
CREATE INDEX IF NOT EXISTS idx_products_fts_gin ON products USING gin(
    to_tsvector('english', name || ' ' || coalesce(description, ''))
);

-- 2. Composite indexes for filtering by category and sorting by price
CREATE INDEX IF NOT EXISTS idx_products_cat_price ON products(category, price);

-- 3. Index for sorting by price alone
CREATE INDEX IF NOT EXISTS idx_products_price_asc ON products(price ASC);
CREATE INDEX IF NOT EXISTS idx_products_price_desc ON products(price DESC);

