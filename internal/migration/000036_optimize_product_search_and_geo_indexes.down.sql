-- Rollback Phase 1 Product Search & Geo Indexes
DROP INDEX IF EXISTS idx_products_trgm_name;
DROP INDEX IF EXISTS idx_products_shop_active;
DROP INDEX IF EXISTS idx_products_category_active;
DROP INDEX IF EXISTS idx_shops_lat_lng;
