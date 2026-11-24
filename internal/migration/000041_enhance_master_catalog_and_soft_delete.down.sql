-- Migration: 000041_enhance_master_catalog_and_soft_delete.down.sql
DROP INDEX IF EXISTS idx_products_shop_barcode_active;
DROP INDEX IF EXISTS idx_products_shop_sku_active;
DROP INDEX IF EXISTS idx_products_shop_active_deleted;
DROP INDEX IF EXISTS idx_master_catalog_barcode_clean;

ALTER TABLE products DROP COLUMN IF EXISTS master_product_id;
ALTER TABLE products DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE master_catalog_products DROP COLUMN IF EXISTS hsn_code;
ALTER TABLE master_catalog_products DROP COLUMN IF EXISTS gst_rate;
ALTER TABLE master_catalog_products DROP COLUMN IF EXISTS is_verified;
ALTER TABLE master_catalog_products DROP COLUMN IF EXISTS deleted_at;
