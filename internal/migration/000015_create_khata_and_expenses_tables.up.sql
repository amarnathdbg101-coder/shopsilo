-- 000015_create_khata_and_expenses_tables.up.sql

-- 1. Customer Khata (Udhar ledger)
CREATE TABLE IF NOT EXISTS customer_khata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_name VARCHAR(150) NOT NULL,
    customer_mobile VARCHAR(20) NOT NULL,
    current_balance NUMERIC(12, 2) DEFAULT 0 CHECK (current_balance >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_shop_customer_mobile UNIQUE (shop_id, customer_mobile)
);

CREATE INDEX IF NOT EXISTS idx_customer_khata_shop ON customer_khata(shop_id);
CREATE INDEX IF NOT EXISTS idx_customer_khata_mobile ON customer_khata(customer_mobile);
CREATE INDEX IF NOT EXISTS idx_customer_khata_balance ON customer_khata(shop_id, current_balance);

-- 2. Khata Transactions (Credit Given vs Payment Received)
CREATE TABLE IF NOT EXISTS khata_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    khata_id UUID NOT NULL REFERENCES customer_khata(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('GIVE_CREDIT', 'RECEIVE_PAYMENT')),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    balance_after NUMERIC(12, 2) NOT NULL CHECK (balance_after >= 0),
    notes TEXT,
    bill_number VARCHAR(100),
    payment_mode VARCHAR(20), -- 'cash', 'upi', etc. for RECEIVE_PAYMENT
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_khata_tx_khata_id ON khata_transactions(khata_id);
CREATE INDEX IF NOT EXISTS idx_khata_tx_shop_id ON khata_transactions(shop_id);
CREATE INDEX IF NOT EXISTS idx_khata_tx_created ON khata_transactions(created_at DESC);

-- 3. Shop Expenses (Dukan ke Roz ke Kharche)
CREATE TABLE IF NOT EXISTS shop_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL, -- rent, electricity, staff_salary, tea_snacks, packaging, other
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    notes TEXT,
    payment_method VARCHAR(20) DEFAULT 'cash',
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop ON shop_expenses(shop_id);
CREATE INDEX IF NOT EXISTS idx_shop_expenses_date ON shop_expenses(shop_id, expense_date);
CREATE INDEX IF NOT EXISTS idx_shop_expenses_category ON shop_expenses(shop_id, category);

-- 4. Shop Timings & Weekly Off columns
ALTER TABLE shops ADD COLUMN IF NOT EXISTS opening_time VARCHAR(10) DEFAULT '09:00';
ALTER TABLE shops ADD COLUMN IF NOT EXISTS closing_time VARCHAR(10) DEFAULT '21:00';
ALTER TABLE shops ADD COLUMN IF NOT EXISTS weekly_off VARCHAR(20) DEFAULT '';
