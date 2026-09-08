-- 000019_ensure_shops_columns.up.sql
-- Ensure all shop columns exist even if shops table was created in an older schema

ALTER TABLE shops
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS category VARCHAR(100),
    ADD COLUMN IF NOT EXISTS phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS address TEXT,
    ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500),
    ADD COLUMN IF NOT EXISTS banners JSONB DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS timing VARCHAR(100),
    ADD COLUMN IF NOT EXISTS is_open BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
