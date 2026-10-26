-- Migration: 000020_create_procurement_items_table.up.sql
-- Cloud Database Persistence for Mandi Procurement & Smart Reorder Checklists

CREATE TABLE IF NOT EXISTS procurement_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL DEFAULT 'General Store',
    qty VARCHAR(100) NOT NULL DEFAULT '1',
    estimated_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    actual_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    dealer_name VARCHAR(255) NOT NULL DEFAULT '',
    dealer_phone VARCHAR(50) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_procurement_items_shop_id ON procurement_items(shop_id);
CREATE INDEX IF NOT EXISTS idx_procurement_items_completed ON procurement_items(shop_id, is_completed);
