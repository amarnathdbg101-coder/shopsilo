-- 000025_add_split_payments_to_pos_bills.down.sql

ALTER TABLE pos_bills
DROP COLUMN IF EXISTS cash_amount,
DROP COLUMN IF EXISTS online_amount,
DROP COLUMN IF EXISTS khata_amount;
