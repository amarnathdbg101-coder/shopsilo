-- 000011_create_shop_reviews_table.up.sql
-- Shop Reviews and Trust Ratings

CREATE TABLE IF NOT EXISTS shop_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment VARCHAR(500),
    is_verified_visitor BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(shop_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_shop_reviews_shop ON shop_reviews(shop_id);
CREATE INDEX IF NOT EXISTS idx_shop_reviews_user ON shop_reviews(user_id);
