-- 000035_add_customer_khata_dual_entry.up.sql
-- Synchronized Dual-Entry Khata & Customer Udhar Passbook Architecture

-- 1. Link customer user account to customer_khata
ALTER TABLE customer_khata
    ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_customer_khata_customer_id ON customer_khata(customer_id);

-- Auto-link existing customer_khata records to registered users by 10-digit mobile number match
UPDATE customer_khata ck
SET customer_id = u.id
FROM users u
WHERE ck.customer_id IS NULL
  AND u.phone IS NOT NULL
  AND LENGTH(REGEXP_REPLACE(u.phone, '[^0-9]', '', 'g')) >= 10
  AND LENGTH(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g')) >= 10
  AND RIGHT(REGEXP_REPLACE(u.phone, '[^0-9]', '', 'g'), 10) = RIGHT(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g'), 10);

-- 2. Enhance khata_transactions with dispute tracking & UPI settlement proof
ALTER TABLE khata_transactions
    ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
    ADD COLUMN IF NOT EXISTS dispute_reason TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS disputed_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS upi_ref_no VARCHAR(50) DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_khata_tx_status ON khata_transactions(khata_id, status);

-- 3. Add UPI ID to shops table for 1-tap customer-to-dukandar settlements
ALTER TABLE shops
    ADD COLUMN IF NOT EXISTS upi_id VARCHAR(100) DEFAULT '';
