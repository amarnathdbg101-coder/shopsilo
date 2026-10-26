-- Migration: 000022_add_discount_text_and_audit_table.up.sql
-- Adds discount_text, min_points_required to store_offers and creates shop_offers_audit table

ALTER TABLE store_offers 
    ADD COLUMN IF NOT EXISTS discount_text VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS min_points_required INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS shop_offers_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offer_id VARCHAR(100) NOT NULL,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    previous_title VARCHAR(200),
    previous_discount_text VARCHAR(100),
    previous_description TEXT,
    new_title VARCHAR(200),
    new_discount_text VARCHAR(100),
    new_description TEXT,
    changed_by_user_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shop_offers_audit_shop ON shop_offers_audit(shop_id, created_at DESC);
