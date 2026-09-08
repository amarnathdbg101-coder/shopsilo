-- 000008_create_shops_table.up.sql
-- Shops table (one user -> one shop)

CREATE TABLE IF NOT EXISTS shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(160) UNIQUE NOT NULL,
    description TEXT,
    category VARCHAR(100),
    phone VARCHAR(20),
    address TEXT,
    logo_url VARCHAR(500),
    banners JSONB DEFAULT '[]',
    timing VARCHAR(100),
    is_open BOOLEAN DEFAULT true,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shops_user ON shops(user_id);
CREATE INDEX IF NOT EXISTS idx_shops_slug ON shops(slug);
CREATE INDEX IF NOT EXISTS idx_shops_category ON shops(category);
CREATE INDEX IF NOT EXISTS idx_shops_name_lower ON shops(LOWER(name));

DO $$ 
BEGIN 
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_products_shop'
    ) THEN 
        ALTER TABLE products 
        ADD CONSTRAINT fk_products_shop 
        FOREIGN KEY (shop_id) REFERENCES shops(id) ON DELETE CASCADE;
    END IF; 
END $$;
