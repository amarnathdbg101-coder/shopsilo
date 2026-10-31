-- 000025_add_search_performance_indexes.down.sql
DROP INDEX IF EXISTS idx_products_attributes_trgm;
DROP INDEX IF EXISTS idx_products_tags_trgm;
DROP FUNCTION IF EXISTS immutable_array_to_string(text[], text);
DROP INDEX IF EXISTS idx_products_tags_gin;
DROP INDEX IF EXISTS idx_shops_address_trgm;
