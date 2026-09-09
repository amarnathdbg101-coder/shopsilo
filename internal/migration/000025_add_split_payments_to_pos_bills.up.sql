-- 000025_add_split_payments_to_pos_bills.up.sql
-- Multi-Tender POS Split Payments (Cash + UPI + Khata)

ALTER TABLE pos_bills
ADD COLUMN IF NOT EXISTS cash_amount DECIMAL(10,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS online_amount DECIMAL(10,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS khata_amount DECIMAL(10,2) DEFAULT 0;

-- Backfill legacy records for existing transactions
UPDATE pos_bills 
SET cash_amount = total_amount 
WHERE payment_method = 'cash' AND cash_amount = 0;

UPDATE pos_bills 
SET online_amount = total_amount 
WHERE payment_method IN ('upi', 'online', 'card') AND online_amount = 0;

UPDATE pos_bills 
SET khata_amount = total_amount 
WHERE payment_method IN ('khata', 'credit') AND khata_amount = 0;
