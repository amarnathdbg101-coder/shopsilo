-- 000040_add_price_visibility_to_products.up.sql
-- Add price visibility toggle for shop owners to hide/show prices to online customers

ALTER TABLE products
ADD COLUMN IF NOT EXISTS is_price_public BOOLEAN DEFAULT true;
