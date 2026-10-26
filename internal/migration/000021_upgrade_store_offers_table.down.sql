ALTER TABLE store_offers 
    DROP COLUMN IF EXISTS discount_type,
    DROP COLUMN IF EXISTS discount_value,
    DROP COLUMN IF EXISTS min_order_amount,
    DROP COLUMN IF EXISTS max_discount,
    DROP COLUMN IF EXISTS applicable_days,
    DROP COLUMN IF EXISTS banner_tag,
    DROP COLUMN IF EXISTS promo_code,
    DROP COLUMN IF EXISTS updated_at;

DROP INDEX IF EXISTS idx_store_offers_shop_days;
