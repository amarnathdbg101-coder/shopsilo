-- 000035_add_customer_khata_dual_entry.down.sql
DROP INDEX IF EXISTS idx_khata_tx_status;
DROP INDEX IF EXISTS idx_customer_khata_customer_id;

ALTER TABLE khata_transactions
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS dispute_reason,
    DROP COLUMN IF EXISTS disputed_at,
    DROP COLUMN IF EXISTS upi_ref_no;

ALTER TABLE customer_khata
    DROP COLUMN IF EXISTS customer_id;

ALTER TABLE shops
    DROP COLUMN IF EXISTS upi_id;
