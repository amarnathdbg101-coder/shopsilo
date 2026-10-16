-- 000015_add_trigram_and_geo_indexes.up.sql
-- 1. Enable PostgreSQL trigram extension for fuzzy & substring ILIKE searching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. Fast GIN Trigram indexes on Products for lightning-fast search
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_sku_trgm ON products USING gin (sku gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_desc_trgm ON products USING gin (description gin_trgm_ops);

-- 3. Geo composite filtered index for nearby shops bounding-box queries
CREATE INDEX IF NOT EXISTS idx_shops_geo_active ON shops (latitude, longitude)
WHERE is_active = true AND status = 'active' AND latitude IS NOT NULL AND longitude IS NOT NULL;

-- 4. Filtered index on inventory for in-stock products
CREATE INDEX IF NOT EXISTS idx_inventory_in_stock ON inventory (product_id)
WHERE (quantity - reserved_quantity) > 0;
