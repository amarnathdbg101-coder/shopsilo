-- 000025_add_search_performance_indexes.up.sql
-- Production Search Optimization for Shops (address) and Products (tags, attributes, text)

-- 1. Ensure pg_trgm extension is active
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. Trigram GIN index on shops(address) for fast locality and area-based search
CREATE INDEX IF NOT EXISTS idx_shops_address_trgm ON shops USING gin (address gin_trgm_ops);

-- 3. Native GIN index on products(tags) for array-based tag lookups
CREATE INDEX IF NOT EXISTS idx_products_tags_gin ON products USING gin (tags);

-- 4. Immutable helper for tags string conversion to support GIN trigram indexing
CREATE OR REPLACE FUNCTION immutable_array_to_string(arr text[], sep text) 
RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ 
    SELECT array_to_string(arr, sep); 
$$;

CREATE INDEX IF NOT EXISTS idx_products_tags_trgm ON products USING gin (immutable_array_to_string(tags, ' ') gin_trgm_ops);

-- 5. Trigram GIN index on products((attributes::text)) to accelerate attribute text searching
CREATE INDEX IF NOT EXISTS idx_products_attributes_trgm ON products USING gin ((attributes::text) gin_trgm_ops);
