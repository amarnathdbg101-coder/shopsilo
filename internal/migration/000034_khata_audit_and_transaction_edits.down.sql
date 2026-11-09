-- 000034_khata_audit_and_transaction_edits.down.sql
DROP INDEX IF EXISTS idx_khata_tx_active;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS is_deleted;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS deleted_by;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS delete_reason;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS original_amount;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS edit_reason;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS modified_by;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS updated_at;

