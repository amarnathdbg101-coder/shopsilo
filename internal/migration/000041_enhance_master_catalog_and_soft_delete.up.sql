-- Migration: 000041_enhance_master_catalog_and_soft_delete.up.sql
-- Enterprise Master Catalog enhancement, Barcode indexing & Soft-Delete architecture

-- 1. Enhance master_catalog_products with GST, HSN, verification and soft-delete
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS trust_score INT DEFAULT 100;
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS scan_count INT DEFAULT 1;
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS hsn_code VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS gst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_master_catalog_barcode_clean 
    ON master_catalog_products (TRIM(LOWER(barcode))) 
    WHERE deleted_at IS NULL;

-- 2. Link store products to master catalog and enforce soft-delete
ALTER TABLE products ADD COLUMN IF NOT EXISTS master_product_id UUID REFERENCES master_catalog_products(id) ON DELETE SET NULL;
ALTER TABLE products ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

-- Performance index for active products excluding soft-deleted items
CREATE INDEX IF NOT EXISTS idx_products_shop_active_deleted 
    ON products(shop_id, is_active) 
    WHERE deleted_at IS NULL;

-- Fast index for SKU and Barcode lookups within a specific shop
CREATE INDEX IF NOT EXISTS idx_products_shop_sku_active 
    ON products(shop_id, TRIM(LOWER(sku))) 
    WHERE is_active = true AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_products_shop_barcode_active 
    ON products(shop_id, TRIM(LOWER(attributes->>'barcode'))) 
    WHERE attributes ? 'barcode' AND is_active = true AND deleted_at IS NULL;
