-- 000021_add_credit_limit_to_khata.up.sql
ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS credit_limit NUMERIC(12, 2) DEFAULT 0 CHECK (credit_limit >= 0);
