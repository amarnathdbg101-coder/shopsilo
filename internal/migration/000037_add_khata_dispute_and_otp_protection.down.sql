DROP INDEX IF EXISTS idx_khata_tx_reversal;

ALTER TABLE khata_transactions
DROP COLUMN IF EXISTS reversal_of_id,
DROP COLUMN IF EXISTS resolved_at,
DROP COLUMN IF EXISTS resolution_notes,
DROP COLUMN IF EXISTS resolution_action;

ALTER TABLE customer_khata
DROP COLUMN IF EXISTS credit_otp_threshold,
DROP COLUMN IF EXISTS credit_otp_required;
