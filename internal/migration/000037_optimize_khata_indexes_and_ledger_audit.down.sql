-- Rollback Phase 1 Khata Performance & Ledger Audit Optimizations
DROP INDEX IF EXISTS idx_customer_khata_shop_phone_digits;
DROP INDEX IF EXISTS idx_khata_tx_khata_created;
ALTER TABLE customer_khata DROP CONSTRAINT IF EXISTS chk_khata_balance_non_negative;
