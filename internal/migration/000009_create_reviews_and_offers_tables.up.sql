-- 000009_create_reviews_and_offers_tables.up.sql
CREATE TABLE IF NOT EXISTS shop_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_shop_user_review UNIQUE (shop_id, user_id)
);

CREATE TABLE IF NOT EXISTS store_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    discount_percent DECIMAL(5, 2),
    promo_code VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
