ALTER TABLE pos_bills
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'completed',
ADD COLUMN IF NOT EXISTS cancellation_reason TEXT,
ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_pos_bills_status ON pos_bills(status);
