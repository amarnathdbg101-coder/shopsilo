-- 1. Enable PostGIS Extension safely if supported by PostgreSQL host
DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS postgis;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;
END;
$$;

-- 2. Safely add spatial GEOGRAPHY column only if postgis geography type is available
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'geography') THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
        ) THEN
            ALTER TABLE shops ADD COLUMN location geography(Point, 4326);
        END IF;
    END IF;
END;
$$;

-- 3. Populate geography from existing latitude and longitude columns safely
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
    ) THEN
        UPDATE shops 
        SET location = ST_SetSRID(ST_MakePoint(longitude::double precision, latitude::double precision), 4326)::geography 
        WHERE latitude IS NOT NULL AND longitude IS NOT NULL AND location IS NULL
          AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180;
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END;
$$;

-- 4. Create GIST Spatial Index safely if location column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_shops_location_gist 
        ON shops USING gist (location) 
        WHERE is_active = true AND status = 'active';
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END;
$$;

-- 5. Trigger function to automatically keep location in sync whenever lat/lng are inserted or updated
-- Hardened with SECURITY DEFINER, explicit search_path, and full exception handling.
CREATE OR REPLACE FUNCTION sync_shop_location()
RETURNS TRIGGER 
SECURITY DEFINER
SET search_path = public, extensions, pg_catalog
AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        BEGIN
            IF NEW.latitude BETWEEN -90 AND 90 AND NEW.longitude BETWEEN -180 AND 180 THEN
                NEW.location := ST_SetSRID(ST_MakePoint(NEW.longitude::double precision, NEW.latitude::double precision), 4326)::geography;
            ELSE
                NEW.location := NULL;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            NEW.location := NULL;
        END;
    ELSE
        BEGIN
            NEW.location := NULL;
        EXCEPTION WHEN OTHERS THEN
            NULL;
        END;
    END IF;
    RETURN NEW;
EXCEPTION WHEN OTHERS THEN
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 6. Attach trigger ONLY IF the 'location' column actually exists in 'shops' table.
-- If 'location' column does not exist, trigger is explicitly DROPPED to prevent record "new" has no field "location" error!
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
    ) THEN
        DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
        CREATE TRIGGER trg_sync_shop_location
        BEFORE INSERT OR UPDATE OF latitude, longitude ON shops
        FOR EACH ROW EXECUTE FUNCTION sync_shop_location();
    ELSE
        DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
    END IF;
END;
$$;


