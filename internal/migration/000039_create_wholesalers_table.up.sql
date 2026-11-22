-- Migration: 000039_create_wholesalers_table.up.sql
-- Cloud Database Persistence for Permanent Mandi Wholesalers & Suppliers Directory

CREATE TABLE IF NOT EXISTS wholesalers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    gstin VARCHAR(50) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wholesalers_shop_id ON wholesalers(shop_id);
CREATE INDEX IF NOT EXISTS idx_wholesalers_name ON wholesalers(shop_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wholesalers_shop_name_unique ON wholesalers(shop_id, LOWER(name));
