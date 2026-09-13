UPDATE categories
SET is_active = true
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

DELETE FROM categories
WHERE slug IN (
    'rice-grains', 'pulses-lentils', 'flour-atta', 'edible-oils', 'spices-masala',
    'sugar-salt', 'packaged-foods', 'biscuits-cookies', 'chips-namkeen', 'tea-coffee',
    'soft-drinks-juices', 'milk-curd', 'butter-cheese', 'bread-bakery', 'eggs',
    'fresh-fruits', 'fresh-vegetables', 'meat-seafood', 'soap-bath', 'hair-care',
    'skin-care', 'oral-care', 'baby-care', 'laundry-care', 'home-cleaning',
    'kitchen-supplies', 'puja-religious', 'stationery-office', 'books-magazines',
    'toys-games', 'mens-clothing', 'womens-clothing', 'kids-clothing', 'footwear',
    'mobile-accessories', 'electronics', 'hardware-electrical', 'pharmacy',
    'vitamins-supplements', 'pet-supplies', 'automotive', 'sports-fitness',
    'gifts-handicrafts'
);