-- 000019_add_shop_id_to_categories.up.sql
-- Enables multi-tenant custom category support per shop with safe global fallbacks.

ALTER TABLE categories ADD COLUMN IF NOT EXISTS shop_id UUID REFERENCES shops(id) ON DELETE CASCADE;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT '📦';
ALTER TABLE categories ADD COLUMN IF NOT EXISTS is_custom BOOLEAN DEFAULT false;

-- Update existing custom flag: if shop_id is not null, it is a custom category
UPDATE categories SET is_custom = true WHERE shop_id IS NOT NULL;

-- Remove global rigid unique constraint on slug so different shops can create categories with the same name
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_slug_key;

-- Multi-tenant partial unique indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_global_slug ON categories(slug) WHERE shop_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_shop_slug ON categories(shop_id, slug) WHERE shop_id IS NOT NULL;

-- Query optimization index for shop category retrieval
CREATE INDEX IF NOT EXISTS idx_categories_shop_lookup ON categories(shop_id, is_active);
