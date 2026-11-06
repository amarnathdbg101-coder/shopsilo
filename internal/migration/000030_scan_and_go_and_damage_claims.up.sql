-- 000030_scan_and_go_and_damage_claims.up.sql
-- Rush-Hour Scan & Go Queue Buster, Company Damage/Expiry Claim Vault, and Reverse Returns Lookup

-- 1. Scan & Go Sessions Table
CREATE TABLE IF NOT EXISTS scan_and_go_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    customer_name VARCHAR(100) NOT NULL DEFAULT 'In-Store Shopper',
    customer_phone VARCHAR(20) NOT NULL DEFAULT '',
    session_code VARCHAR(10) UNIQUE NOT NULL,
    qr_token VARCHAR(64) UNIQUE NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'VERIFIED_AT_COUNTER', 'CONVERTED_TO_BILL', 'EXPIRED', 'CANCELLED')),
    total_items INT NOT NULL DEFAULT 0,
    subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    expires_at TIMESTAMPTZ NOT NULL,
    converted_bill_id UUID REFERENCES pos_bills(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_scan_and_go_shop_status ON scan_and_go_sessions(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_scan_and_go_code ON scan_and_go_sessions(session_code);
CREATE INDEX IF NOT EXISTS idx_scan_and_go_token ON scan_and_go_sessions(qr_token);

-- 2. Scan & Go Cart Items Table
CREATE TABLE IF NOT EXISTS scan_and_go_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES scan_and_go_sessions(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    product_name VARCHAR(255) NOT NULL,
    barcode VARCHAR(100) NOT NULL DEFAULT '',
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    total_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_scan_and_go_items_sess ON scan_and_go_items(session_id);

-- 3. Damage & Expiry Claims Table
CREATE TABLE IF NOT EXISTS damage_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    product_name VARCHAR(255) NOT NULL,
    barcode VARCHAR(100) NOT NULL DEFAULT '',
    batch_number VARCHAR(100) NOT NULL DEFAULT '',
    expiry_date DATE,
    quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_cost DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    total_loss DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    claim_type VARCHAR(50) NOT NULL DEFAULT 'EXPIRED' CHECK (claim_type IN ('EXPIRED', 'TRANSIT_DAMAGE', 'LEAKAGE_SPILL', 'RAT_PEST_BITE', 'FACTORY_DEFECT', 'OTHER')),
    distributor_name VARCHAR(255) NOT NULL DEFAULT '',
    distributor_phone VARCHAR(50) NOT NULL DEFAULT '',
    sales_rep_name VARCHAR(255) NOT NULL DEFAULT '',
    photo_url TEXT NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING_CLAIM' CHECK (status IN ('PENDING_CLAIM', 'SUBMITTED_TO_SR', 'REPLACED', 'CREDIT_NOTE_ISSUED', 'REJECTED')),
    resolution_notes TEXT NOT NULL DEFAULT '',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_damage_claims_shop_distributor ON damage_claims(shop_id, distributor_name);
CREATE INDEX IF NOT EXISTS idx_damage_claims_shop_status ON damage_claims(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_damage_claims_product ON damage_claims(product_id);

-- 4. Reverse Returns Search Indexes
CREATE INDEX IF NOT EXISTS idx_pos_bills_phone ON pos_bills(customer_phone) WHERE customer_phone IS NOT NULL AND customer_phone != '';
CREATE INDEX IF NOT EXISTS idx_pos_bill_items_product ON pos_bill_items(product_id);

