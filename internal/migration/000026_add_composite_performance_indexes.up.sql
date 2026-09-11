-- 000026_add_composite_performance_indexes.up.sql
-- High-performance production composite indexes for low latency queries

-- 1. Fast customer previous basket lookup & customer bill history
CREATE INDEX IF NOT EXISTS idx_pos_bills_customer_phone ON pos_bills(shop_id, customer_phone, created_at DESC);

-- 2. Fast category product filtering with active status
CREATE INDEX IF NOT EXISTS idx_products_category_active ON products(category_id, is_active);

-- 3. Fast khata transaction type aggregation & passbook statement query
CREATE INDEX IF NOT EXISTS idx_khata_tx_shop_type_created ON khata_transactions(shop_id, type, created_at DESC);

-- 4. Fast daily cash/upi expense calculation
CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop_method_date ON shop_expenses(shop_id, payment_method, expense_date);

-- 5. Fast reservation cleanup of expired active reservations
CREATE INDEX IF NOT EXISTS idx_reservations_status_expires ON reservations(status, expires_at);
