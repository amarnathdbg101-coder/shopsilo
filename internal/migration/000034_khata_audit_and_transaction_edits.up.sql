-- 000034_khata_audit_and_transaction_edits.up.sql
-- Adds audit columns for reliable editing and soft-deletion of Khata customer credit entries.

ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS deleted_by VARCHAR(100);
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS delete_reason TEXT;
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS original_amount DECIMAL(10, 2);
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS edit_reason TEXT;
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS modified_by VARCHAR(100);
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_khata_tx_active ON khata_transactions(khata_id, is_deleted, created_at DESC);

