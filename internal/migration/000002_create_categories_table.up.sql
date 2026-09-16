-- 000002_create_categories_table.up.sql
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

INSERT INTO categories (name, slug, description, icon) VALUES
('Kirana & Grocery', 'kirana-grocery', 'Daily essentials, rice, flour, oil, pulses and spices', '🛒'),
('Dairy & Breakfast', 'dairy-breakfast', 'Milk, curd, butter, paneer, eggs and bread', '🥛'),
('Snacks & Beverages', 'snacks-beverages', 'Biscuits, namkeen, cold drinks, tea and coffee', '🍪'),
('Fruits & Vegetables', 'fruits-vegetables', 'Fresh local farm fruits and green vegetables', '🥦'),
('Personal Care', 'personal-care', 'Soaps, shampoos, toothpaste, skincare and haircare', '🧼'),
('Household Cleaning', 'household-cleaning', 'Detergents, floor cleaners, dishwash and mosquito repellents', '🧹'),
('Clothing & Apparel', 'clothing-apparel', 'Menswear, womenswear, kids wear and innerwear', '👕'),
('Footwear', 'footwear', 'Casual shoes, sandals, slippers and formal shoes', '👟'),
('Electronics & Accessories', 'electronics-accessories', 'Mobile chargers, earphones, cables, bulbs and batteries', '⚡'),
('Pharmacy & Healthcare', 'pharmacy-healthcare', 'Over-the-counter medicines, first aid, vitamins and supplements', '💊')
ON CONFLICT (slug) DO NOTHING;
