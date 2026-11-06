-- ==============================================================================
-- 000027_create_locations_and_variants.up.sql
-- Multi-Store, Godowns, Location Inventory & Apparel/Cosmetic Variant Matrix
-- ==============================================================================

-- 1. LOCATIONS (Main Retail Store, Godown #1, Basement Stockroom, etc.)
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    type VARCHAR(30) NOT NULL DEFAULT 'RETAIL_STORE' CHECK (type IN ('RETAIL_STORE', 'WAREHOUSE', 'GODOWN', 'OUTLET')),
    address TEXT,
    contact_person VARCHAR(100),
    contact_phone VARCHAR(20),
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(shop_id, code)
);

CREATE INDEX IF NOT EXISTS idx_locations_shop ON locations(shop_id, is_active);
CREATE INDEX IF NOT EXISTS idx_locations_default ON locations(shop_id) WHERE is_default = true;

-- Auto-provision a default "Main Showroom" location for all existing shops
INSERT INTO locations (shop_id, name, code, type, is_default, is_active)
SELECT id, 'Main Showroom', 'MAIN', 'RETAIL_STORE', true, true
FROM shops
ON CONFLICT (shop_id, code) DO NOTHING;

-- 2. APPAREL & COSMETIC PRODUCT VARIANTS (Child SKUs: Size, Color, Shade, Batch, Expiry)
CREATE TABLE IF NOT EXISTS product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    barcode VARCHAR(100),
    title VARCHAR(200) NOT NULL,
    size VARCHAR(30),
    color VARCHAR(50),
    color_hex VARCHAR(10),
    shade_code VARCHAR(50),
    batch_number VARCHAR(50),
    expiry_date DATE,
    price DECIMAL(10, 2) NOT NULL,
    cost_price DECIMAL(10, 2) DEFAULT 0,
    compare_price DECIMAL(10, 2) DEFAULT 0,
    floor_price DECIMAL(10, 2) DEFAULT 0,
    weight DECIMAL(8, 2) DEFAULT 0,
    image_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(shop_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_product_variants_product ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS idx_product_variants_shop ON product_variants(shop_id, is_active);
CREATE INDEX IF NOT EXISTS idx_product_variants_barcode ON product_variants(shop_id, barcode) WHERE barcode IS NOT NULL AND barcode != '';
CREATE INDEX IF NOT EXISTS idx_product_variants_size ON product_variants(shop_id, size) WHERE size IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_product_variants_color ON product_variants(shop_id, color) WHERE color IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_product_variants_expiry ON product_variants(shop_id, expiry_date) WHERE expiry_date IS NOT NULL;

-- 3. LOCATION INVENTORY (Physical Stock per Location & per Variant with Row-Level Lock guards)
CREATE TABLE IF NOT EXISTS location_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    low_stock_threshold INT NOT NULL DEFAULT 5,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Partial unique indexes to support both simple products (variant_id IS NULL) and variants
CREATE UNIQUE INDEX IF NOT EXISTS idx_loc_inv_variant_uniq 
ON location_inventory(location_id, product_id, variant_id) 
WHERE variant_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_loc_inv_simple_uniq 
ON location_inventory(location_id, product_id) 
WHERE variant_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_loc_inv_location_prod ON location_inventory(location_id, product_id);
CREATE INDEX IF NOT EXISTS idx_loc_inv_variant ON location_inventory(variant_id);

-- Migrate existing legacy inventory data into the default location_inventory (Zero Data Loss)
INSERT INTO location_inventory (shop_id, location_id, product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
SELECT p.shop_id, l.id, inv.product_id, inv.quantity, inv.reserved_quantity, inv.low_stock_threshold, inv.updated_at
FROM inventory inv
JOIN products p ON inv.product_id = p.id
JOIN locations l ON l.shop_id = p.shop_id AND l.is_default = true
ON CONFLICT (location_id, product_id) WHERE variant_id IS NULL 
DO UPDATE SET quantity = EXCLUDED.quantity, reserved_quantity = EXCLUDED.reserved_quantity, updated_at = NOW();

-- 4. STOCK TRANSFERS (Godown <-> Store State Machine: REQUESTED -> DISPATCHED -> IN_TRANSIT -> RECEIVED -> RECONCILED)
CREATE TABLE IF NOT EXISTS stock_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    transfer_number VARCHAR(50) NOT NULL,
    from_location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
    to_location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'DISPATCHED', 'IN_TRANSIT', 'RECEIVED', 'RECONCILED', 'CANCELLED')),
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    received_by UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(shop_id, transfer_number)
);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_shop_status ON stock_transfers(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_from_loc ON stock_transfers(from_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_to_loc ON stock_transfers(to_location_id);

CREATE TABLE IF NOT EXISTS stock_transfer_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    requested_qty INT NOT NULL CHECK (requested_qty > 0),
    sent_qty INT NOT NULL DEFAULT 0 CHECK (sent_qty >= 0),
    received_qty INT NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_transfer ON stock_transfer_items(transfer_id);
