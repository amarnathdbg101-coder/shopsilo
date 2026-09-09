-- 000021_add_credit_limit_to_khata.down.sql
ALTER TABLE customer_khata DROP COLUMN IF EXISTS credit_limit;
