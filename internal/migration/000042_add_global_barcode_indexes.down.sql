-- Migration: 000042_add_global_barcode_indexes.down.sql
DROP INDEX IF EXISTS idx_products_global_barcode_active;
DROP INDEX IF EXISTS idx_products_global_sku_active;
