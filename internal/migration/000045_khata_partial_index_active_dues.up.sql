-- 000045_khata_partial_index_active_dues.up.sql
-- Partial B-Tree index for active debtors (current_balance > 0) to accelerate Khata list & Vasooli queries.

CREATE INDEX IF NOT EXISTS idx_khata_active_dues
    ON customer_khata (shop_id, current_balance DESC, updated_at DESC)
    WHERE current_balance > 0;
