-- 000040_add_price_visibility_to_products.down.sql
ALTER TABLE products
DROP COLUMN IF EXISTS is_price_public;
