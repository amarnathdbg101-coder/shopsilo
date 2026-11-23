-- 000040_create_retail_intelligence_tables.up.sql

DO $$ 
BEGIN 
    CREATE EXTENSION IF NOT EXISTS vector; 
EXCEPTION WHEN OTHERS THEN 
    RAISE NOTICE 'pgvector extension not installed or permission denied, semantic text search fallback will be used';
END $$;

-- 1. Product Intelligence (ABC-XYZ, Velocity, Elasticity, Dead Capital, Attributes)
CREATE TABLE IF NOT EXISTS product_intelligence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    search_keywords TEXT NOT NULL DEFAULT '',
    abc_tier VARCHAR(5) NOT NULL DEFAULT 'C',          -- 'A', 'B', 'C' (Revenue contribution)
    xyz_tier VARCHAR(5) NOT NULL DEFAULT 'Z',          -- 'X', 'Y', 'Z' (Demand volatility)
    velocity_daily DECIMAL(10, 4) NOT NULL DEFAULT 0.0,
    days_of_supply INT NOT NULL DEFAULT 999,
    days_since_last_sale INT NOT NULL DEFAULT 0,
    is_dead_stock BOOLEAN NOT NULL DEFAULT false,
    carrying_cost_monthly DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    price_elasticity DECIMAL(6, 2) NOT NULL DEFAULT -1.0,
    is_price_elastic BOOLEAN NOT NULL DEFAULT true,
    clearance_stage INT NOT NULL DEFAULT 0,            -- 0: Normal, 1: Mild promo, 2: Deep clearance, 3: Dead stock liquidation
    recommended_action VARCHAR(255) NOT NULL DEFAULT '',
    recommended_min_price DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_product_intelligence_product UNIQUE (product_id)
);

CREATE INDEX IF NOT EXISTS idx_product_intelligence_shop ON product_intelligence(shop_id, abc_tier, xyz_tier);
CREATE INDEX IF NOT EXISTS idx_product_intelligence_dead ON product_intelligence(shop_id, is_dead_stock);
CREATE INDEX IF NOT EXISTS idx_product_intelligence_product ON product_intelligence(product_id);

-- 2. Customer Psychology & Sensitivity (RFM, Bargaining tendency, Category affinities)
CREATE TABLE IF NOT EXISTS customer_psychology (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_phone VARCHAR(50) NOT NULL,
    customer_name VARCHAR(200) NOT NULL DEFAULT '',
    rfm_segment VARCHAR(30) NOT NULL DEFAULT 'REGULAR', -- VIP_GOLD, LOYAL_SILVER, OCCASIONAL, BARGAINER
    total_lifetime_spend DECIMAL(12, 2) NOT NULL DEFAULT 0.0,
    total_orders INT NOT NULL DEFAULT 0,
    avg_order_value DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    bargain_tendency_score DECIMAL(3, 2) NOT NULL DEFAULT 0.50, -- 0.00 (Firm) to 1.00 (Heavy Bargainer)
    max_discount_affinity DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    preferred_categories TEXT[] NOT NULL DEFAULT '{}',
    preferred_fabrics_or_brands TEXT[] NOT NULL DEFAULT '{}',
    last_purchase_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_customer_psychology_shop_phone UNIQUE (shop_id, customer_phone)
);

CREATE INDEX IF NOT EXISTS idx_customer_psychology_shop ON customer_psychology(shop_id, rfm_segment);
CREATE INDEX IF NOT EXISTS idx_customer_psychology_phone ON customer_psychology(shop_id, customer_phone);

-- 3. Daily Prescriptions (The Morning 3-Action Queue - Cognitive Load Governor)
CREATE TABLE IF NOT EXISTS daily_prescriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    urgency_tier VARCHAR(30) NOT NULL DEFAULT 'MEDIUM', -- URGENT_CASH, STOCKOUT_CRITICAL, KHATA_RECOVERY
    category VARCHAR(50) NOT NULL DEFAULT 'GENERAL',    -- DEAD_STOCK, REORDER, PROMO, NEGOTIATION
    headline VARCHAR(255) NOT NULL,
    explanation TEXT NOT NULL,
    financial_impact_inr DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    executable_action_type VARCHAR(50) NOT NULL,        -- APPLY_DISCOUNT, SEND_PO, SEND_REMINDER, CLEAR_STOCK
    action_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_executed BOOLEAN NOT NULL DEFAULT false,
    executed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_daily_prescriptions_shop ON daily_prescriptions(shop_id, is_executed, created_at DESC);
