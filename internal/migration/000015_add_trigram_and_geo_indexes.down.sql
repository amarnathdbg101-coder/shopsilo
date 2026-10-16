-- 000015_add_trigram_and_geo_indexes.down.sql
DROP INDEX IF EXISTS idx_inventory_in_stock;
DROP INDEX IF EXISTS idx_shops_geo_active;
DROP INDEX IF EXISTS idx_products_desc_trgm;
DROP INDEX IF EXISTS idx_products_sku_trgm;
DROP INDEX IF EXISTS idx_products_name_trgm;
