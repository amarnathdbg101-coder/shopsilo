-- 000017_seed_default_categories.down.sql
DELETE FROM categories WHERE slug IN (
    'kirana-grocery', 'snacks-drinks', 'dairy-bread-eggs', 'personal-care',
    'fruits-vegetables', 'electronics-mobile', 'clothing-fashion',
    'pharmacy-wellness', 'household-cleaning', 'general-store'
);
