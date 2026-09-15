-- Enable pg_trgm for high performance fuzzy / trigram ILIKE indexing
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes on product fields for instant sub-millisecond search
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_desc_trgm ON products USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_sku_trgm ON products USING gin (sku gin_trgm_ops);

-- GIN trigram indexes on shop fields
CREATE INDEX IF NOT EXISTS idx_shops_name_trgm ON shops USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_shops_category_trgm ON shops USING gin (category gin_trgm_ops);
