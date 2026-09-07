-- 000013_create_loyalty_and_returns_table.up.sql
-- Customer Loyalty Points, In-Store Product Returns, and VIP Store Offers

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS loyalty_points INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS product_returns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    reservation_id UUID REFERENCES reservations(id) ON DELETE SET NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    refund_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS store_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    discount_text VARCHAR(100) NOT NULL,
    min_points_required INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_returns_shop ON product_returns(shop_id);
CREATE INDEX IF NOT EXISTS idx_returns_product ON product_returns(product_id);
CREATE INDEX IF NOT EXISTS idx_offers_shop ON store_offers(shop_id);
CREATE INDEX IF NOT EXISTS idx_offers_active ON store_offers(is_active) WHERE is_active = true;
