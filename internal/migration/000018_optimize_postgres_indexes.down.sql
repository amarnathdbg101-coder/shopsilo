-- 000018_optimize_postgres_indexes.down.sql
DROP INDEX IF EXISTS idx_products_featured;
DROP INDEX IF EXISTS idx_products_shop_cat_active;
DROP INDEX IF EXISTS idx_reservations_active_expires;
DROP INDEX IF EXISTS idx_reservations_pickup_code;
DROP INDEX IF EXISTS idx_customer_khata_phone_last10;
DROP INDEX IF EXISTS idx_users_phone_last10;
DROP INDEX IF EXISTS idx_customer_khata_balance;
DROP INDEX IF EXISTS idx_pos_bills_shop_status_date;
DROP INDEX IF EXISTS idx_pos_bills_customer_phone;
DROP INDEX IF EXISTS idx_products_attributes_gin;
DROP INDEX IF EXISTS idx_products_shop_sku_upper;
DROP INDEX IF EXISTS idx_products_shop_barcode;
