ALTER TABLE khata_transactions DROP COLUMN IF EXISTS parchi_image_url;
ALTER TABLE khata_transactions DROP COLUMN IF EXISTS items_summary;
ALTER TABLE customer_khata DROP COLUMN IF EXISTS promise_to_pay_date;
ALTER TABLE customer_khata DROP COLUMN IF EXISTS installment_target;
