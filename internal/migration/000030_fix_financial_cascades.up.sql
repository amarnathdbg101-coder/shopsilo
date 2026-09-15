-- 000030_fix_financial_cascades.up.sql
-- Fix critical bug: Prevent cascading deletes from wiping out financial history (Bills, Returns, Reservations)

DO $$
DECLARE
    fk_name text;
BEGIN
    -- 1. Fix pos_bill_items -> products (Keep bill item if product is hard-deleted)
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'pos_bill_items' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE pos_bill_items DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE pos_bill_items ALTER COLUMN product_id DROP NOT NULL;
    ALTER TABLE pos_bill_items ADD CONSTRAINT pos_bill_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

    -- 2. Fix reservations -> users (Keep shop's completed reservation history if user deletes account)
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'reservations' AND column_name = 'user_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE reservations DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE reservations ALTER COLUMN user_id DROP NOT NULL;
    ALTER TABLE reservations ADD CONSTRAINT reservations_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

    -- 3. Fix reservations -> products (Keep shop's completed reservation history if product is hard-deleted)
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'reservations' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE reservations DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE reservations ALTER COLUMN product_id DROP NOT NULL;
    ALTER TABLE reservations ADD CONSTRAINT reservations_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

    -- 4. Fix product_returns -> products (Keep refund/return history if product is hard-deleted)
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'product_returns' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE product_returns DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE product_returns ALTER COLUMN product_id DROP NOT NULL;
    ALTER TABLE product_returns ADD CONSTRAINT product_returns_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

    -- 5. Fix product_bargain_deals -> products (Keep bargaining history)
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'product_bargain_deals' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE product_bargain_deals DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE product_bargain_deals ALTER COLUMN product_id DROP NOT NULL;
    ALTER TABLE product_bargain_deals ADD CONSTRAINT product_bargain_deals_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;

END $$;
