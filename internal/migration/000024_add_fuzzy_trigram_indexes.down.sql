-- 000024_add_fuzzy_trigram_indexes.down.sql
DROP INDEX IF EXISTS idx_shops_name_trgm;
DROP INDEX IF EXISTS idx_shops_desc_trgm;
DROP INDEX IF EXISTS idx_master_catalog_name_trgm;
DROP INDEX IF EXISTS idx_master_catalog_brand_trgm;
