-- Down Migration: Revert PostGIS spatial columns, index, and trigger
DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
DROP FUNCTION IF EXISTS sync_shop_location();
DROP INDEX IF EXISTS idx_shops_location_gist;
ALTER TABLE shops DROP COLUMN IF EXISTS location;

