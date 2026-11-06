-- 000031_batches_and_micro_plans.up.sql
-- Batch Expiry Radar, Near-Expiry Auto-Markdown Engine, and Khata Daily Micro-Installment Settle

-- 1. Product Batches Table
CREATE TABLE IF NOT EXISTS product_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    batch_number VARCHAR(100) NOT NULL,
    mfg_date DATE,
    expiry_date DATE NOT NULL,
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    cost_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    original_selling_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    current_selling_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    markdown_percent DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    is_markdown_applied BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'NEAR_EXPIRY', 'EXPIRED', 'SOLD_OUT')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_product_batches_shop_expiry ON product_batches(shop_id, expiry_date ASC);
CREATE INDEX IF NOT EXISTS idx_product_batches_product ON product_batches(product_id, status);
CREATE INDEX IF NOT EXISTS idx_product_batches_status ON product_batches(shop_id, status);

-- 2. Khata Micro-Installment Plans Table
CREATE TABLE IF NOT EXISTS khata_micro_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    customer_phone VARCHAR(20) NOT NULL,
    customer_name VARCHAR(100) NOT NULL DEFAULT 'Khata Customer',
    total_debt DECIMAL(10, 2) NOT NULL CHECK (total_debt > 0),
    daily_amount DECIMAL(10, 2) NOT NULL CHECK (daily_amount > 0),
    frequency VARCHAR(20) NOT NULL DEFAULT 'DAILY' CHECK (frequency IN ('DAILY', 'WEEKLY', 'MONTHLY')),
    total_installments INT NOT NULL CHECK (total_installments > 0),
    paid_installments INT NOT NULL DEFAULT 0,
    total_paid_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    remaining_balance DECIMAL(10, 2) NOT NULL,
    next_due_date DATE NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'PAUSED', 'DEFAULTED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_khata_micro_plans_shop ON khata_micro_plans(shop_id, status);
CREATE INDEX IF NOT EXISTS idx_khata_micro_plans_phone ON khata_micro_plans(customer_phone);

-- 3. Khata Micro Payments Log Table
CREATE TABLE IF NOT EXISTS khata_micro_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES khata_micro_plans(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    installment_number INT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
    payment_mode VARCHAR(20) NOT NULL DEFAULT 'CASH' CHECK (payment_mode IN ('CASH', 'UPI', 'QR')),
    transaction_reference VARCHAR(100) DEFAULT '',
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_khata_micro_payments_plan ON khata_micro_payments(plan_id, installment_number);

