-- Migration: 000017_create_master_catalog_products.up.sql
-- Enterprise Global Master Barcode Catalog with Crowdsourcing & GS1 India Support

CREATE TABLE IF NOT EXISTS master_catalog_products (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode       VARCHAR(64)  NOT NULL,
    name          VARCHAR(255) NOT NULL,
    brand         VARCHAR(100) NOT NULL DEFAULT '',
    category      VARCHAR(100) NOT NULL DEFAULT 'General Store',
    unit          VARCHAR(50)  NOT NULL DEFAULT 'piece',
    mrp           NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    selling_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    description   TEXT,
    image_url     TEXT,
    source        VARCHAR(50)  NOT NULL DEFAULT 'crowdsourced',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Primary O(log n) B-Tree unique index on barcode for instant sub-millisecond lookup
CREATE UNIQUE INDEX IF NOT EXISTS idx_master_catalog_barcode
    ON master_catalog_products (TRIM(barcode));

-- Secondary index for brand-based searches
CREATE INDEX IF NOT EXISTS idx_master_catalog_brand
    ON master_catalog_products (LOWER(TRIM(brand)));

-- Secondary index for category browsing
CREATE INDEX IF NOT EXISTS idx_master_catalog_category
    ON master_catalog_products (LOWER(TRIM(category)));
