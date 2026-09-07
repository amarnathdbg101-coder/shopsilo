-- 000016_optimize_production_indexes.up.sql
-- High-performance composite indexes for production query speed & low latency

-- 1. POS Bills & Analytics speedup
CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_created ON pos_bills(shop_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pos_bill_items_product ON pos_bill_items(product_id);

-- 2. Products query speedup
CREATE INDEX IF NOT EXISTS idx_products_shop_active ON products(shop_id, is_active);
CREATE INDEX IF NOT EXISTS idx_products_name_lower ON products(LOWER(name));

-- 3. Reservations speedup
CREATE INDEX IF NOT EXISTS idx_reservations_shop_status ON reservations(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_reservations_user_status ON reservations(user_id, status);

-- 4. Shop location and status composite index
CREATE INDEX IF NOT EXISTS idx_shops_active_open ON shops(is_active, is_open);
