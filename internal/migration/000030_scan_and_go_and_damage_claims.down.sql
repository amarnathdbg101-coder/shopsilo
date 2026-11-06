-- 000030_scan_and_go_and_damage_claims.down.sql
DROP INDEX IF EXISTS idx_pos_bill_items_product;
DROP INDEX IF EXISTS idx_pos_bills_phone;
DROP TABLE IF EXISTS damage_claims;
DROP TABLE IF EXISTS scan_and_go_items;
DROP TABLE IF EXISTS scan_and_go_sessions;

