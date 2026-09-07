-- 000009_add_location_and_contact_to_shops.up.sql
-- Add location coordinates, city, pincode, and WhatsApp contact for in-store discovery

ALTER TABLE shops
    ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS city VARCHAR(100),
    ADD COLUMN IF NOT EXISTS pincode VARCHAR(20),
    ADD COLUMN IF NOT EXISTS whatsapp_number VARCHAR(20);

CREATE INDEX IF NOT EXISTS idx_shops_city ON shops(LOWER(city));
CREATE INDEX IF NOT EXISTS idx_shops_pincode ON shops(pincode);
CREATE INDEX IF NOT EXISTS idx_shops_coordinates ON shops(latitude, longitude);
