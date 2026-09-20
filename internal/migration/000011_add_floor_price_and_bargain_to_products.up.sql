-- 000011_add_floor_price_and_bargain_to_products.up.sql
ALTER TABLE products ADD COLUMN IF NOT EXISTS floor_price DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE products ADD COLUMN IF NOT EXISTS allow_bargain BOOLEAN DEFAULT false;
ALTER TABLE products ADD COLUMN IF NOT EXISTS is_price_public BOOLEAN DEFAULT true;
