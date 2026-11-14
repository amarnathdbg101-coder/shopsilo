-- Up Migration: 000038_create_app_versions_ota.up.sql
-- Table to manage Over-The-Air (OTA) in-app updates for ShopSilo Merchant POS and Customer apps.

CREATE TABLE IF NOT EXISTS app_versions (
    id SERIAL PRIMARY KEY,
    app_type VARCHAR(50) NOT NULL DEFAULT 'merchant_pos', -- 'merchant_pos', 'customer_app'
    platform VARCHAR(20) NOT NULL DEFAULT 'android',       -- 'android', 'ios', 'windows'
    version_code INT NOT NULL,                             -- e.g. 101, 102 (matches Flutter build-number)
    version_name VARCHAR(50) NOT NULL,                     -- e.g. '1.0.1' (matches Flutter build-name)
    min_supported_version_code INT NOT NULL DEFAULT 1,     -- Versions below this will be forced to update
    apk_url TEXT NOT NULL,                                 -- Direct download URL or relative /downloads/path.apk
    file_size_bytes BIGINT NOT NULL DEFAULT 0,             -- Total size in bytes for download progress bar
    release_notes JSONB NOT NULL DEFAULT '[]'::jsonb,      -- Array of strings e.g. ["Feature 1", "Bugfix 2"]
    is_force_update BOOLEAN NOT NULL DEFAULT FALSE,        -- Mandatory update flag
    is_active BOOLEAN NOT NULL DEFAULT TRUE,               -- Soft delete / toggle flag
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Fast lookup index for active version check queries
CREATE INDEX IF NOT EXISTS idx_app_versions_lookup 
ON app_versions (app_type, platform, is_active, version_code DESC);

-- Seed initial merchant pos version
INSERT INTO app_versions (
    app_type, platform, version_code, version_name, min_supported_version_code,
    apk_url, file_size_bytes, release_notes, is_force_update, is_active
) VALUES (
    'merchant_pos',
    'android',
    101,
    '1.0.1',
    100,
    '/downloads/shopsilo-merchant-v1.0.1.apk',
    28500000,
    '["⚡ नया लाइव मार्जिन फ्लोर गार्ड (Profit Guard)", "🗄️ कैश ड्रॉवर ₹3,000+ सेफ ड्रॉप अलर्ट", "🚀 रश-आवर 10s पिकअप PIN सत्यापन"]'::jsonb,
    false,
    true
) ON CONFLICT DO NOTHING;
