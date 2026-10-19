-- Migration: 000016_create_pos_parked_bills_table.up.sql
-- High-Performance Live Multi-Basket Cart Parking for Assisted Selling

CREATE TABLE IF NOT EXISTS pos_parked_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    label VARCHAR(100) NOT NULL DEFAULT '',
    customer_phone VARCHAR(20) NOT NULL DEFAULT '',
    cart_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pos_parked_bills_shop_id ON pos_parked_bills(shop_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pos_parked_bills_shop_label ON pos_parked_bills(shop_id, UPPER(TRIM(label))) WHERE label != '';
