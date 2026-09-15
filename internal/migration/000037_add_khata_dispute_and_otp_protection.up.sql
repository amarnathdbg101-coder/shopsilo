-- Migration 37: Add khata dispute resolution and credit OTP protection
ALTER TABLE customer_khata
ADD COLUMN IF NOT EXISTS credit_otp_required BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS credit_otp_threshold NUMERIC(10,2) NOT NULL DEFAULT 1000.00;

ALTER TABLE khata_transactions
ADD COLUMN IF NOT EXISTS resolution_action VARCHAR(30),
ADD COLUMN IF NOT EXISTS resolution_notes TEXT,
ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS reversal_of_id UUID REFERENCES khata_transactions(id);

CREATE INDEX IF NOT EXISTS idx_khata_tx_reversal ON khata_transactions(reversal_of_id);
