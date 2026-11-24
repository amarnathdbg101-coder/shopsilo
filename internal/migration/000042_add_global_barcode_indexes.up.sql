-- Migration: 000042_add_global_barcode_indexes.up.sql
-- High-speed B-Tree expression indexes for universal SKU and barcode crowdsourced resolution across all shops

-- Global SKU lookup index across active store products
CREATE INDEX IF NOT EXISTS idx_products_global_sku_active 
    ON products (TRIM(LOWER(sku))) 
    WHERE is_active = true AND deleted_at IS NULL;

-- Global JSONB Barcode lookup index across active store products
CREATE INDEX IF NOT EXISTS idx_products_global_barcode_active 
    ON products (TRIM(LOWER(attributes->>'barcode'))) 
    WHERE attributes ? 'barcode' AND is_active = true AND deleted_at IS NULL;
