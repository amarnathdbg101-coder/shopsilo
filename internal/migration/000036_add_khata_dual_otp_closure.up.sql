-- Add closure status, OTP and audit columns to customer_khata
ALTER TABLE customer_khata
ADD COLUMN IF NOT EXISTS closure_status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
ADD COLUMN IF NOT EXISTS closure_otp VARCHAR(10),
ADD COLUMN IF NOT EXISTS closure_requested_by VARCHAR(30),
ADD COLUMN IF NOT EXISTS closure_requested_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP;

-- Index for instant multi-shop phone lookups
CREATE INDEX IF NOT EXISTS idx_customer_khata_mobile_clean
ON customer_khata (customer_mobile);

-- Index for closure status
CREATE INDEX IF NOT EXISTS idx_customer_khata_closure_status
ON customer_khata (closure_status);
