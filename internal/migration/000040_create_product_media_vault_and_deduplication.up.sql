-- 000040_create_product_media_vault_and_deduplication.up.sql
-- Smart Product Image Vault & Universal Asset Deduplication Engine

CREATE TABLE IF NOT EXISTS product_media_vault (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_code VARCHAR(100) NOT NULL, -- Barcode, EAN, or normalized SKU/Slug
    content_hash VARCHAR(64) UNIQUE NOT NULL,  -- SHA-256 hex digest of image file bytes
    image_url TEXT NOT NULL,            -- Cloudflare R2 / Storage public URL
    original_filename VARCHAR(255),
    mime_type VARCHAR(50),
    file_size_bytes BIGINT DEFAULT 0,
    uploader_shop_id UUID REFERENCES shops(id) ON DELETE SET NULL,
    is_verified_master BOOLEAN DEFAULT false,
    reference_count INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_vault_code ON product_media_vault(product_code);
CREATE INDEX IF NOT EXISTS idx_media_vault_hash ON product_media_vault(content_hash);
CREATE INDEX IF NOT EXISTS idx_media_vault_url ON product_media_vault(image_url);
CREATE INDEX IF NOT EXISTS idx_media_vault_shop ON product_media_vault(uploader_shop_id);
CREATE INDEX IF NOT EXISTS idx_media_vault_verified ON product_media_vault(is_verified_master) WHERE is_verified_master = true;
