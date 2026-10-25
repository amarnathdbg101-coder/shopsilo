-- 000018_optimize_postgres_indexes.up.sql
-- Enterprise PostgreSQL Production Performance Optimization Migration
-- Optimizes POS barcode lookups, 10-digit Indian phone searches, reservation cleanups,
-- storefront category filtering, and daily reconciliation aggregations.

-- 1. POS Lightning-Fast Barcode & SKU Lookup (Sub-millisecond counter scanning)
CREATE INDEX IF NOT EXISTS idx_products_shop_barcode 
    ON products(shop_id, (attributes->>'barcode')) 
    WHERE attributes ? 'barcode';

CREATE INDEX IF NOT EXISTS idx_products_shop_sku_upper 
    ON products(shop_id, UPPER(TRIM(sku))) 
    WHERE sku IS NOT NULL AND sku != '';

CREATE INDEX IF NOT EXISTS idx_products_attributes_gin 
    ON products USING gin (attributes jsonb_path_ops);

-- 2. Customer POS Bill History & Shop Sales Filtering
CREATE INDEX IF NOT EXISTS idx_pos_bills_customer_phone 
    ON pos_bills(customer_phone, created_at DESC) 
    WHERE customer_phone IS NOT NULL AND customer_phone != '';

CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_status_date 
    ON pos_bills(shop_id, status, created_at DESC);

-- 3. Khata Fast Ledger Lookups & Normalized Indian 10-Digit Phone Indexes
CREATE INDEX IF NOT EXISTS idx_customer_khata_balance 
    ON customer_khata(shop_id, current_balance DESC) 
    WHERE current_balance > 0;

CREATE INDEX IF NOT EXISTS idx_users_phone_last10 
    ON users (RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10));

CREATE INDEX IF NOT EXISTS idx_customer_khata_phone_last10 
    ON customer_khata (shop_id, RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10));

-- 4. In-Store Counter Pickup Pass & Background Expiration Index
CREATE INDEX IF NOT EXISTS idx_reservations_pickup_code 
    ON reservations(shop_id, pickup_code) 
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_reservations_active_expires 
    ON reservations(expires_at) 
    WHERE status = 'active';

-- 5. Storefront Browsing, Category Filter & Featured Deals Indexes
CREATE INDEX IF NOT EXISTS idx_products_shop_cat_active 
    ON products(shop_id, category_id, is_active) 
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_products_featured 
    ON products(shop_id, is_featured, created_at DESC) 
    WHERE is_featured = true AND is_active = true;
