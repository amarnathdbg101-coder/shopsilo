-- 000019_add_shop_id_to_categories.down.sql
DROP INDEX IF EXISTS idx_categories_shop_lookup;
DROP INDEX IF EXISTS idx_categories_shop_slug;
DROP INDEX IF EXISTS idx_categories_global_slug;

DELETE FROM categories WHERE shop_id IS NOT NULL;

ALTER TABLE categories DROP COLUMN IF EXISTS is_custom;
ALTER TABLE categories DROP COLUMN IF EXISTS icon;
ALTER TABLE categories DROP COLUMN IF EXISTS shop_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
