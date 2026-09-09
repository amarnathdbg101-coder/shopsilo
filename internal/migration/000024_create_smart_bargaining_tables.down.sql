-- 000024_create_smart_bargaining_tables.down.sql

DROP TABLE IF EXISTS product_bargain_deals;

ALTER TABLE products
DROP COLUMN IF EXISTS floor_price,
DROP COLUMN IF EXISTS allow_bargain;
