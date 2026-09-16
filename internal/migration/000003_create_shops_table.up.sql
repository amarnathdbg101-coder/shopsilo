-- 000003_create_shops_table.up.sql
CREATE TABLE IF NOT EXISTS shops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    description TEXT,
    category VARCHAR(100),
    phone VARCHAR(20) NOT NULL,
    address TEXT NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    city VARCHAR(100) NOT NULL,
    pincode VARCHAR(10) NOT NULL,
    whatsapp_number VARCHAR(20),
    logo_url TEXT,
    banners JSONB DEFAULT '[]'::jsonb,
    timing VARCHAR(100),
    opening_time VARCHAR(10) DEFAULT '09:00',
    closing_time VARCHAR(10) DEFAULT '21:00',
    weekly_off VARCHAR(20),
    is_open BOOLEAN DEFAULT true,
    is_active BOOLEAN DEFAULT true,
    status VARCHAR(30) DEFAULT 'active',
    flagged_count INT DEFAULT 0,
    suspension_reason TEXT,
    creation_ip VARCHAR(50),
    creation_user_agent TEXT,
    device_fingerprint VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_shops_user ON shops(user_id);
CREATE INDEX IF NOT EXISTS idx_shops_slug ON shops(slug);
CREATE INDEX IF NOT EXISTS idx_shops_city ON shops(city);
CREATE INDEX IF NOT EXISTS idx_shops_status ON shops(status, is_active);
