-- 000013_create_loyalty_and_returns_table.down.sql

DROP TABLE IF EXISTS store_offers;
DROP TABLE IF EXISTS product_returns;

ALTER TABLE users
    DROP COLUMN IF EXISTS loyalty_points;
