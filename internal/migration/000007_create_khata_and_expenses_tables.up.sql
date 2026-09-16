-- 000007_create_khata_and_expenses_tables.up.sql
CREATE TABLE IF NOT EXISTS customer_khata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_phone VARCHAR(20) NOT NULL,
    customer_name VARCHAR(100) NOT NULL,
    current_balance DECIMAL(10, 2) NOT NULL DEFAULT 0,
    credit_limit DECIMAL(10, 2) DEFAULT 5000.00,
    otp_protection BOOLEAN DEFAULT false,
    promise_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_shop_customer_phone UNIQUE (shop_id, customer_phone)
);
CREATE INDEX IF NOT EXISTS idx_khata_shop_mobile ON customer_khata(shop_id, customer_phone);

CREATE TABLE IF NOT EXISTS khata_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    khata_id UUID NOT NULL REFERENCES customer_khata(id) ON DELETE CASCADE,
    type VARCHAR(30) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    balance_after DECIMAL(10, 2) NOT NULL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_khata_tx_khata ON khata_transactions(khata_id, created_at DESC);

CREATE TABLE IF NOT EXISTS daily_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    notes TEXT,
    expense_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_expenses_shop_date ON daily_expenses(shop_id, expense_date DESC);
