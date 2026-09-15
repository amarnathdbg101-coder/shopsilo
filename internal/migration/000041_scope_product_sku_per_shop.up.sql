-- 000041_scope_product_sku_per_shop.up.sql
-- Scope SKU uniqueness to individual shops (Multi-tenant standard)
-- Drops global table-level unique constraint on SKU so multiple shops can share common Barcodes/SKUs

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_sku_key;
DROP INDEX IF EXISTS idx_products_sku;
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_shop_sku ON products (shop_id, UPPER(sku));
CREATE INDEX IF NOT EXISTS idx_products_sku ON products (sku);
