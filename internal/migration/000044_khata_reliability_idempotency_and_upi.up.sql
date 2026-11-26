-- 000044_khata_reliability_idempotency_and_upi.up.sql
-- Adds DB-level idempotency key for khata transactions, accurate overdue tracking (oldest_unpaid_at),
-- and unique UTR/UPI reference guard per khata to prevent duplicate or unverified UPI settlements.

ALTER TABLE khata_transactions
    ADD COLUMN IF NOT EXISTS client_tx_id VARCHAR(100);

-- Unique index per shop for non-empty client_tx_id so retries never create duplicate entries
CREATE UNIQUE INDEX IF NOT EXISTS idx_khata_tx_shop_client_tx_id
    ON khata_transactions (shop_id, client_tx_id)
    WHERE client_tx_id IS NOT NULL AND client_tx_id <> '';

-- Prevent duplicate non-rejected UPI UTR submissions on the same khata
CREATE UNIQUE INDEX IF NOT EXISTS idx_khata_tx_unique_upi_ref
    ON khata_transactions (khata_id, LOWER(TRIM(upi_ref_no)))
    WHERE upi_ref_no IS NOT NULL AND TRIM(upi_ref_no) <> '' AND COALESCE(status, 'CONFIRMED') <> 'REJECTED';

-- Track when the current unpaid debt cycle began (not reset by partial payments or limit edits)
ALTER TABLE customer_khata
    ADD COLUMN IF NOT EXISTS oldest_unpaid_at TIMESTAMP;

-- Backfill existing indebted accounts safely
UPDATE customer_khata
SET oldest_unpaid_at = COALESCE(oldest_unpaid_at, updated_at, created_at)
WHERE current_balance > 0 AND oldest_unpaid_at IS NULL;
