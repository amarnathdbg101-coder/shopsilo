-- 000006_create_orders_table.down.sql
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS order_status;
