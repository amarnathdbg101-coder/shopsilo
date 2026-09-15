-- 000041_scope_product_sku_per_shop.down.sql
DROP INDEX IF EXISTS idx_products_shop_sku;
ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);
