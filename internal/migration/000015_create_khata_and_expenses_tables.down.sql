-- 000015_create_khata_and_expenses_tables.down.sql

DROP TABLE IF EXISTS khata_transactions;
DROP TABLE IF EXISTS customer_khata;
DROP TABLE IF EXISTS shop_expenses;

ALTER TABLE shops DROP COLUMN IF EXISTS opening_time;
ALTER TABLE shops DROP COLUMN IF EXISTS closing_time;
ALTER TABLE shops DROP COLUMN IF EXISTS weekly_off;
