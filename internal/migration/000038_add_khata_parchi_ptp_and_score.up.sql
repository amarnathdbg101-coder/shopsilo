-- Add parchi photo and items summary to khata transactions
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS parchi_image_url TEXT;
ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS items_summary TEXT;

-- Add Promise to Pay (PTP) date and installment target to customer khata
ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS promise_to_pay_date TIMESTAMPTZ;
ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS installment_target NUMERIC(12,2) DEFAULT 0.00;
