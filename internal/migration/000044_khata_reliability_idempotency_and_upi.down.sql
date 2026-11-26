-- 000044_khata_reliability_idempotency_and_upi.down.sql
DROP INDEX IF EXISTS idx_khata_tx_shop_client_tx_id;
DROP INDEX IF EXISTS idx_khata_tx_unique_upi_ref;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS client_tx_id;
ALTER TABLE customer_khata DROP COLUMN IF EXISTS oldest_unpaid_at;
