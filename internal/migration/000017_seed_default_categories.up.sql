-- 000017_seed_default_categories.up.sql
-- Seed standard retail categories for Indian shops & commerce

INSERT INTO categories (id, name, slug, description, is_active)
VALUES
    (gen_random_uuid(), 'Kirana & Grocery', 'kirana-grocery', 'Daily staples, atta, dal, rice, oil and spices', true),
    (gen_random_uuid(), 'Snacks & Drinks', 'snacks-drinks', 'Biscuits, chips, namkeen, cold drinks and juices', true),
    (gen_random_uuid(), 'Dairy, Bread & Eggs', 'dairy-bread-eggs', 'Milk, curd, butter, cheese, paneer and bread', true),
    (gen_random_uuid(), 'Personal Care', 'personal-care', 'Soaps, shampoo, toothpaste, skin and hair care', true),
    (gen_random_uuid(), 'Fruits & Vegetables', 'fruits-vegetables', 'Fresh seasonal fruits and fresh vegetables', true),
    (gen_random_uuid(), 'Electronics & Mobile', 'electronics-mobile', 'Mobile accessories, chargers, earphones and appliances', true),
    (gen_random_uuid(), 'Clothing & Fashion', 'clothing-fashion', 'Men, women and kids daily wear and accessories', true),
    (gen_random_uuid(), 'Pharmacy & Wellness', 'pharmacy-wellness', 'First-aid, daily medicines, vitamins and supplements', true),
    (gen_random_uuid(), 'Household & Cleaning', 'household-cleaning', 'Detergent, dishwash, cleaners and pooja essentials', true),
    (gen_random_uuid(), 'General Store', 'general-store', 'Stationery, plastics, gifts and miscellaneous items', true)
ON CONFLICT (slug) DO NOTHING;
