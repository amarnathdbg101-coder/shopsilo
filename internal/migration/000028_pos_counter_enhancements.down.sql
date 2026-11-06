-- 000028_pos_counter_enhancements.down.sql
DROP TABLE IF EXISTS shop_hotkeys CASCADE;
DROP INDEX IF EXISTS idx_pos_bill_items_variant;
ALTER TABLE pos_bill_items DROP COLUMN IF EXISTS variant_id;
ALTER TABLE product_variants DROP COLUMN IF EXISTS min_margin_percent;
ALTER TABLE products DROP COLUMN IF EXISTS min_margin_percent;

