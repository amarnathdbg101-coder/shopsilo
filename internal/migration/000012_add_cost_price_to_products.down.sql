-- 000012_add_cost_price_to_products.down.sql

ALTER TABLE products
    DROP COLUMN IF EXISTS cost_price;
