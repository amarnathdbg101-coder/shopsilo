-- 000018_add_attributes_to_products.up.sql
-- Add flexible JSONB attributes for custom product specifications
-- e.g. brand, model, size, color, gender, season, material, etc. (MongoDB inside PostgreSQL)

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS attributes JSONB DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_products_attributes ON products USING GIN (attributes);
