-- 000011_add_floor_price_and_bargain_to_products.down.sql
ALTER TABLE products DROP COLUMN IF EXISTS floor_price;
ALTER TABLE products DROP COLUMN IF EXISTS allow_bargain;
ALTER TABLE products DROP COLUMN IF EXISTS is_price_public;
