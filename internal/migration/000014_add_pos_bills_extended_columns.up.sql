-- 000014_add_pos_bills_extended_columns.up.sql
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS customer_user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS subtotal DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS total_cost DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cash_amount DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS online_amount DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS khata_amount DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;
ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP;
ALTER TABLE pos_bills ALTER COLUMN cashier_user_id DROP NOT NULL;

ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS product_sku VARCHAR(100) DEFAULT '';
ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS unit_cost DECIMAL(10, 2) DEFAULT 0;
ALTER TABLE pos_bill_items ALTER COLUMN product_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pos_bills_bill_number ON pos_bills(bill_number);
CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_status ON pos_bills(shop_id, status);
