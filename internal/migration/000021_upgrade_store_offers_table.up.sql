-- Migration: 000021_upgrade_store_offers_table.up.sql
-- Day-wise Scheduling, Dynamic Discount Types, and Real-Time Storefront Offers

ALTER TABLE store_offers 
    ADD COLUMN IF NOT EXISTS discount_type VARCHAR(20) NOT NULL DEFAULT 'percentage',
    ADD COLUMN IF NOT EXISTS discount_value DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS min_order_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS max_discount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS applicable_days VARCHAR(100) NOT NULL DEFAULT 'all',
    ADD COLUMN IF NOT EXISTS banner_tag VARCHAR(50) NOT NULL DEFAULT '🔥 Special Offer',
    ADD COLUMN IF NOT EXISTS promo_code VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_store_offers_shop_days ON store_offers(shop_id, is_active, applicable_days);
