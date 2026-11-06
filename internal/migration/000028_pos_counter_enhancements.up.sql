-- 000028_pos_counter_enhancements.up.sql
-- Milestone 1: POS Financial Shield, Live Margin Sentinel & Fast-Grid Hotkeys

-- 1. Add minimum margin percentage to products and variants
ALTER TABLE products ADD COLUMN IF NOT EXISTS min_margin_percent DECIMAL(5, 2) DEFAULT 10.00;
ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS min_margin_percent DECIMAL(5, 2) DEFAULT 10.00;

-- 2. Add variant_id to pos_bill_items for exact variant audit
ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_pos_bill_items_variant ON pos_bill_items(variant_id) WHERE variant_id IS NOT NULL;

-- 3. Fast-Grid Hotkeys table (Rush-Hour Zero-Scan 1-Tap Counter Grid)
CREATE TABLE IF NOT EXISTS shop_hotkeys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    display_name VARCHAR(100) NOT NULL,
    color_hex VARCHAR(10) NOT NULL DEFAULT '#2563EB',
    sort_order INT NOT NULL DEFAULT 0,
    time_slot VARCHAR(20) NOT NULL DEFAULT 'ALL_DAY' CHECK (time_slot IN ('ALL_DAY', 'MORNING', 'EVENING')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shop_hotkeys_lookup ON shop_hotkeys(shop_id, time_slot, is_active, sort_order ASC);

