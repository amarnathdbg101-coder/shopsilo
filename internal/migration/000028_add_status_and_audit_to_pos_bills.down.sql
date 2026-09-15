ALTER TABLE pos_bills
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS cancellation_reason,
DROP COLUMN IF EXISTS cancelled_at;
