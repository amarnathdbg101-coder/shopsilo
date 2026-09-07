-- 000014_create_pos_sales_table.up.sql
-- Counter Billing POS (Point of Sale) and Digital Receipts

CREATE TABLE IF NOT EXISTS pos_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    bill_number VARCHAR(50) UNIQUE NOT NULL,
    customer_phone VARCHAR(20),
    customer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    subtotal DECIMAL(10,2) NOT NULL,
    discount_amount DECIMAL(10,2) DEFAULT 0,
    total_amount DECIMAL(10,2) NOT NULL,
    total_cost DECIMAL(10,2) DEFAULT 0,
    payment_method VARCHAR(20) NOT NULL DEFAULT 'cash', -- 'cash', 'upi', 'card'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pos_bill_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id UUID NOT NULL REFERENCES pos_bills(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_name VARCHAR(200) NOT NULL,
    product_sku VARCHAR(100),
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10,2) NOT NULL,
    unit_cost DECIMAL(10,2) DEFAULT 0,
    total_price DECIMAL(10,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pos_bills_shop ON pos_bills(shop_id);
CREATE INDEX IF NOT EXISTS idx_pos_bills_created ON pos_bills(created_at);
CREATE INDEX IF NOT EXISTS idx_pos_bill_items_bill ON pos_bill_items(bill_id);
