-- 000026_add_composite_performance_indexes.down.sql

DROP INDEX IF EXISTS idx_pos_bills_customer_phone;
DROP INDEX IF EXISTS idx_products_category_active;
DROP INDEX IF EXISTS idx_khata_tx_shop_type_created;
DROP INDEX IF EXISTS idx_shop_expenses_shop_method_date;
DROP INDEX IF EXISTS idx_reservations_status_expires;
