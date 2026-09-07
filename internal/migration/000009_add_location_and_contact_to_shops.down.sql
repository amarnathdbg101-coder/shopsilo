-- 000009_add_location_and_contact_to_shops.down.sql

DROP INDEX IF EXISTS idx_shops_coordinates;
DROP INDEX IF EXISTS idx_shops_pincode;
DROP INDEX IF EXISTS idx_shops_city;

ALTER TABLE shops
    DROP COLUMN IF EXISTS whatsapp_number,
    DROP COLUMN IF EXISTS pincode,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS longitude,
    DROP COLUMN IF EXISTS latitude;
