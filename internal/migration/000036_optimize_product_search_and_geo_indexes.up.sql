-- Phase 1 Product Performance & Search Index Optimizations

-- 1. Ensure pg_trgm extension is active for fast fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. GIN Trigram Index on product names for < 2ms typo-tolerant fuzzy matching
CREATE INDEX IF NOT EXISTS idx_products_trgm_name ON products USING GIN (name gin_trgm_ops);

-- 3. Composite B-Tree Indexes for active shop products and active category products
CREATE INDEX IF NOT EXISTS idx_products_shop_active ON products (shop_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_products_category_active ON products (category_id, is_active) WHERE is_active = true;

-- 4. B-Tree Index on shop coordinates for fast distance filtering
CREATE INDEX IF NOT EXISTS idx_shops_lat_lng ON shops (latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
