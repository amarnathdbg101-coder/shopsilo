-- Migration: 000023_create_shop_expenses_table.up.sql
-- Create operational store expenses table and performance indexes

CREATE TABLE IF NOT EXISTS shop_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    notes TEXT,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'cash',
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop_date ON shop_expenses(shop_id, expense_date DESC);
CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop_created ON shop_expenses(shop_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_shop_expenses_category ON shop_expenses(shop_id, category);
