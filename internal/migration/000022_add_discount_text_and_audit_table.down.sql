-- Migration: 000022_add_discount_text_and_audit_table.down.sql

DROP TABLE IF EXISTS shop_offers_audit CASCADE;

ALTER TABLE store_offers 
    DROP COLUMN IF EXISTS discount_text,
    DROP COLUMN IF EXISTS min_points_required;
