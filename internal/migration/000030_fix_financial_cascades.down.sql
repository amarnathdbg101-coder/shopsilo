-- 000030_fix_financial_cascades.down.sql

DO $$
DECLARE
    fk_name text;
BEGIN
    -- 1. pos_bill_items -> products
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'pos_bill_items' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE pos_bill_items DROP CONSTRAINT ' || fk_name;
    END IF;
    -- Note: Can't easily set NOT NULL if there are nulls, so we skip it.
    ALTER TABLE pos_bill_items ADD CONSTRAINT pos_bill_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

    -- 2. reservations -> users
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'reservations' AND column_name = 'user_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE reservations DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE reservations ADD CONSTRAINT reservations_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

    -- 3. reservations -> products
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'reservations' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE reservations DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE reservations ADD CONSTRAINT reservations_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

    -- 4. product_returns -> products
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'product_returns' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE product_returns DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE product_returns ADD CONSTRAINT product_returns_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

    -- 5. product_bargain_deals -> products
    SELECT constraint_name INTO fk_name FROM information_schema.key_column_usage WHERE table_name = 'product_bargain_deals' AND column_name = 'product_id' LIMIT 1;
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE product_bargain_deals DROP CONSTRAINT ' || fk_name;
    END IF;
    ALTER TABLE product_bargain_deals ADD CONSTRAINT product_bargain_deals_product_id_fkey FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;
END $$;
