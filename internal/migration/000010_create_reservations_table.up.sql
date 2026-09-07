-- 000010_create_reservations_table.up.sql
-- In-Store Product Reservation / Item Hold

DO $$ BEGIN
    CREATE TYPE reservation_status AS ENUM ('active', 'completed', 'cancelled', 'expired');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_number VARCHAR(50) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    pickup_code VARCHAR(10) NOT NULL,
    status reservation_status DEFAULT 'active',
    expires_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reservations_user ON reservations(user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_shop ON reservations(shop_id);
CREATE INDEX IF NOT EXISTS idx_reservations_product ON reservations(product_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status);
CREATE INDEX IF NOT EXISTS idx_reservations_pickup_code ON reservations(pickup_code);
CREATE INDEX IF NOT EXISTS idx_reservations_expires_at ON reservations(expires_at);
