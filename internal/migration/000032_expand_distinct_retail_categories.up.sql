-- Replace broad category buckets with distinct, searchable retail categories.

INSERT INTO categories (id, name, slug, description, is_active)
VALUES
    (gen_random_uuid(), 'Rice & Grains', 'rice-grains', 'Rice, wheat, millets, oats and other grains', true),
    (gen_random_uuid(), 'Pulses & Lentils', 'pulses-lentils', 'Dal, rajma, chana, peas and lentils', true),
    (gen_random_uuid(), 'Flour & Atta', 'flour-atta', 'Atta, maida, besan and baking flour', true),
    (gen_random_uuid(), 'Edible Oils', 'edible-oils', 'Cooking oils, ghee and vanaspati', true),
    (gen_random_uuid(), 'Spices & Masala', 'spices-masala', 'Whole spices, powders and masala blends', true),
    (gen_random_uuid(), 'Sugar & Salt', 'sugar-salt', 'Sugar, salt, jaggery and sweeteners', true),
    (gen_random_uuid(), 'Packaged Foods', 'packaged-foods', 'Ready-to-cook, canned and packaged foods', true),
    (gen_random_uuid(), 'Biscuits & Cookies', 'biscuits-cookies', 'Biscuits, cookies and rusks', true),
    (gen_random_uuid(), 'Chips & Namkeen', 'chips-namkeen', 'Chips, namkeen and savoury snacks', true),
    (gen_random_uuid(), 'Tea & Coffee', 'tea-coffee', 'Tea, coffee, premixes and hot beverages', true),
    (gen_random_uuid(), 'Soft Drinks & Juices', 'soft-drinks-juices', 'Cold drinks, juices, water and energy drinks', true),
    (gen_random_uuid(), 'Milk & Curd', 'milk-curd', 'Milk, curd and dairy beverages', true),
    (gen_random_uuid(), 'Butter & Cheese', 'butter-cheese', 'Butter, cheese, paneer and spreads', true),
    (gen_random_uuid(), 'Bread & Bakery', 'bread-bakery', 'Bread, buns, cakes and bakery products', true),
    (gen_random_uuid(), 'Eggs', 'eggs', 'Eggs and egg-based products', true),
    (gen_random_uuid(), 'Fresh Fruits', 'fresh-fruits', 'Fresh seasonal fruits', true),
    (gen_random_uuid(), 'Fresh Vegetables', 'fresh-vegetables', 'Fresh vegetables and leafy greens', true),
    (gen_random_uuid(), 'Meat & Seafood', 'meat-seafood', 'Meat, chicken, fish and seafood', true),
    (gen_random_uuid(), 'Soap & Bath', 'soap-bath', 'Bath soaps, body wash and bath essentials', true),
    (gen_random_uuid(), 'Hair Care', 'hair-care', 'Shampoo, conditioner, oil and hair styling', true),
    (gen_random_uuid(), 'Skin Care', 'skin-care', 'Creams, lotions, face care and sunscreen', true),
    (gen_random_uuid(), 'Oral Care', 'oral-care', 'Toothpaste, brushes and mouth care', true),
    (gen_random_uuid(), 'Baby Care', 'baby-care', 'Diapers, baby food and baby essentials', true),
    (gen_random_uuid(), 'Laundry Care', 'laundry-care', 'Detergent, fabric care and stain removers', true),
    (gen_random_uuid(), 'Home Cleaning', 'home-cleaning', 'Floor, toilet, dish and surface cleaners', true),
    (gen_random_uuid(), 'Kitchen Supplies', 'kitchen-supplies', 'Kitchen tools, containers and consumables', true),
    (gen_random_uuid(), 'Puja & Religious', 'puja-religious', 'Puja items, incense and religious essentials', true),
    (gen_random_uuid(), 'Stationery & Office', 'stationery-office', 'Pens, paper, notebooks and office supplies', true),
    (gen_random_uuid(), 'Books & Magazines', 'books-magazines', 'Books, newspapers and magazines', true),
    (gen_random_uuid(), 'Toys & Games', 'toys-games', 'Toys, games and children entertainment', true),
    (gen_random_uuid(), 'Mens Clothing', 'mens-clothing', 'Mens clothing and daily wear', true),
    (gen_random_uuid(), 'Womens Clothing', 'womens-clothing', 'Womens clothing and daily wear', true),
    (gen_random_uuid(), 'Kids Clothing', 'kids-clothing', 'Kids clothing and school wear', true),
    (gen_random_uuid(), 'Footwear', 'footwear', 'Shoes, sandals and slippers', true),
    (gen_random_uuid(), 'Mobile Accessories', 'mobile-accessories', 'Chargers, cables, cases and earphones', true),
    (gen_random_uuid(), 'Electronics', 'electronics', 'Small appliances and consumer electronics', true),
    (gen_random_uuid(), 'Hardware & Electrical', 'hardware-electrical', 'Tools, bulbs, wires and electrical supplies', true),
    (gen_random_uuid(), 'Pharmacy', 'pharmacy', 'Daily medicines and first-aid products', true),
    (gen_random_uuid(), 'Vitamins & Supplements', 'vitamins-supplements', 'Vitamins, nutrition and wellness products', true),
    (gen_random_uuid(), 'Pet Supplies', 'pet-supplies', 'Pet food, hygiene and accessories', true),
    (gen_random_uuid(), 'Automotive', 'automotive', 'Vehicle care, accessories and essentials', true),
    (gen_random_uuid(), 'Sports & Fitness', 'sports-fitness', 'Sports goods and fitness accessories', true),
    (gen_random_uuid(), 'Gifts & Handicrafts', 'gifts-handicrafts', 'Gifts, decor and handmade products', true)
ON CONFLICT (slug) DO NOTHING;

UPDATE categories
SET is_active = false
WHERE slug IN (
    'kirana-grocery',
    'snacks-drinks',
    'dairy-bread-eggs',
    'personal-care',
    'fruits-vegetables',
    'electronics-mobile',
    'clothing-fashion',
    'pharmacy-wellness',
    'household-cleaning'
);