-- 000024_create_smart_bargaining_tables.up.sql
-- Smart Dynamic Bargaining & Offer Negotiation Engine (Bhav-Taav)

ALTER TABLE products
ADD COLUMN IF NOT EXISTS floor_price DECIMAL(10,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS allow_bargain BOOLEAN DEFAULT true;

CREATE TABLE IF NOT EXISTS product_bargain_deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_phone VARCHAR(20) NOT NULL,
    customer_name VARCHAR(100),
    deal_code VARCHAR(10) NOT NULL,
    offered_price DECIMAL(10,2) NOT NULL,
    agreed_price DECIMAL(10,2) NOT NULL,
    bundle_quantity INT DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'accepted', -- 'accepted', 'counter_offered', 'redeemed', 'expired'
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bargain_deals_lookup ON product_bargain_deals(deal_code, product_id);
CREATE INDEX IF NOT EXISTS idx_bargain_deals_shop_status ON product_bargain_deals(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_bargain_deals_phone ON product_bargain_deals(customer_phone);
