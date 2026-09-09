-- 000022_create_pos_parked_bills.up.sql
CREATE TABLE IF NOT EXISTS pos_parked_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    label VARCHAR(100),
    customer_phone VARCHAR(20),
    cart_data JSONB NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pos_parked_bills_shop ON pos_parked_bills(shop_id);
CREATE INDEX IF NOT EXISTS idx_pos_parked_bills_created ON pos_parked_bills(created_at);
