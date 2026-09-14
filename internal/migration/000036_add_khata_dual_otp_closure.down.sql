DROP INDEX IF EXISTS idx_customer_khata_closure_status;
DROP INDEX IF EXISTS idx_customer_khata_mobile_clean;

ALTER TABLE customer_khata
DROP COLUMN IF EXISTS closed_at,
DROP COLUMN IF EXISTS closure_requested_at,
DROP COLUMN IF EXISTS closure_requested_by,
DROP COLUMN IF EXISTS closure_otp,
DROP COLUMN IF EXISTS closure_status;
