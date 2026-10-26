-- 000024_add_fuzzy_trigram_indexes.up.sql
-- Enterprise Trigram Indexes for Typo-Tolerant Fuzzy Search (pg_trgm)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 1. Shops trigram indexes
CREATE INDEX IF NOT EXISTS idx_shops_name_trgm ON shops USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_shops_desc_trgm ON shops USING gin (description gin_trgm_ops);

-- 2. Master Catalog trigram indexes
CREATE INDEX IF NOT EXISTS idx_master_catalog_name_trgm ON master_catalog_products USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_master_catalog_brand_trgm ON master_catalog_products USING gin (brand gin_trgm_ops);

-- 3. Products trigram indexes (idempotent verification)
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_sku_trgm ON products USING gin (sku gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_desc_trgm ON products USING gin (description gin_trgm_ops);
