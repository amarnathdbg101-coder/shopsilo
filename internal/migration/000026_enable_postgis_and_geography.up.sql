-- 1. Enable PostGIS Extension (Standard Linux VPS / PostgreSQL extension)
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. Add spatial GEOGRAPHY column (Point type with standard WGS 84 SRID 4326)
ALTER TABLE shops ADD COLUMN IF NOT EXISTS location geography(Point, 4326);

-- 3. Populate geography from existing latitude and longitude columns
UPDATE shops 
SET location = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL AND location IS NULL;

-- 4. Create GIST Spatial Index (R-Tree 2D spatial indexing for microsecond lookups)
CREATE INDEX IF NOT EXISTS idx_shops_location_gist 
ON shops USING gist (location) 
WHERE is_active = true AND status = 'active';

-- 5. Trigger to automatically keep location in sync whenever lat/lng are inserted or updated
CREATE OR REPLACE FUNCTION sync_shop_location()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.location := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    ELSE
        NEW.location := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
CREATE TRIGGER trg_sync_shop_location
BEFORE INSERT OR UPDATE OF latitude, longitude ON shops
FOR EACH ROW EXECUTE FUNCTION sync_shop_location();

