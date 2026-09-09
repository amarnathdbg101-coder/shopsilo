-- 000023_create_product_stock_alerts.up.sql
CREATE TABLE IF NOT EXISTS product_stock_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_phone VARCHAR(20) NOT NULL,
    customer_name VARCHAR(100),
    notified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_product_customer_alert UNIQUE (product_id, customer_phone)
);

CREATE INDEX IF NOT EXISTS idx_stock_alerts_shop ON product_stock_alerts(shop_id);
CREATE INDEX IF NOT EXISTS idx_stock_alerts_product ON product_stock_alerts(product_id);
