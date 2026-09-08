-- 000008_create_shops_table.down.sql
ALTER TABLE products DROP CONSTRAINT IF EXISTS fk_products_shop;
DROP TABLE IF EXISTS shops CASCADE;
