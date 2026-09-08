-- 000018_add_attributes_to_products.down.sql
DROP INDEX IF EXISTS idx_products_attributes;
ALTER TABLE products DROP COLUMN IF EXISTS attributes;
