-- 000006_create_pos_sales_tables.up.sql
CREATE TABLE IF NOT EXISTS pos_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    cashier_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    bill_number VARCHAR(50) UNIQUE NOT NULL,
    customer_phone VARCHAR(20),
    customer_name VARCHAR(100),
    total_amount DECIMAL(10, 2) NOT NULL,
    payment_method VARCHAR(30) NOT NULL DEFAULT 'cash',
    split_cash_amount DECIMAL(10, 2) DEFAULT 0,
    split_upi_amount DECIMAL(10, 2) DEFAULT 0,
    status VARCHAR(30) NOT NULL DEFAULT 'completed',
    audit_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_date ON pos_bills(shop_id, created_at DESC);

CREATE TABLE IF NOT EXISTS pos_bill_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id UUID NOT NULL REFERENCES pos_bills(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    product_name VARCHAR(200) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pos_bill_items_bill ON pos_bill_items(bill_id);
