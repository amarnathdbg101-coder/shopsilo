-- 000016_optimize_production_indexes.down.sql

DROP INDEX IF EXISTS idx_pos_bills_shop_created;
DROP INDEX IF EXISTS idx_pos_bill_items_product;
DROP INDEX IF EXISTS idx_products_shop_active;
DROP INDEX IF EXISTS idx_products_name_lower;
DROP INDEX IF EXISTS idx_reservations_shop_status;
DROP INDEX IF EXISTS idx_reservations_user_status;
DROP INDEX IF EXISTS idx_shops_active_open;
