-- 000012_add_cost_price_to_products.up.sql
-- Add wholesale cost price to products for retail store profit tracking

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS cost_price DECIMAL(10,2) DEFAULT 0;
