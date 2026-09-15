-- Link products with matching categories

-- Milk & Curd
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'milk-curd' LIMIT 1)
WHERE (name ILIKE '%milk%' OR name ILIKE '%doodh%' OR name ILIKE '%curd%' OR name ILIKE '%dahi%')
  AND category_id IS NULL;

-- Butter & Cheese
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'butter-cheese' LIMIT 1)
WHERE (name ILIKE '%butter%' OR name ILIKE '%cheese%' OR name ILIKE '%paneer%' OR name ILIKE '%makhan%')
  AND category_id IS NULL;

-- Edible Oils
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'edible-oils' LIMIT 1)
WHERE (name ILIKE '%oil%' OR name ILIKE '%mustard%' OR name ILIKE '%sunflower%' OR name ILIKE '%ghee%' OR name ILIKE '%refined%')
  AND name NOT ILIKE '%hair oil%' AND name NOT ILIKE '%coconut oil%'
  AND category_id IS NULL;

-- Hair Care
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'hair-care' LIMIT 1)
WHERE (name ILIKE '%hair oil%' OR name ILIKE '%coconut oil%' OR name ILIKE '%shampoo%' OR name ILIKE '%parachute%')
  AND category_id IS NULL;

-- Sugar & Salt
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'sugar-salt' LIMIT 1)
WHERE (name ILIKE '%salt%' OR name ILIKE '%sugar%' OR name ILIKE '%cheeni%' OR name ILIKE '%namak%')
  AND category_id IS NULL;

-- Tea & Coffee
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'tea-coffee' LIMIT 1)
WHERE (name ILIKE '%tea%' OR name ILIKE '%coffee%' OR name ILIKE '%chai%' OR name ILIKE '%red label%')
  AND category_id IS NULL;

-- Biscuits & Cookies
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'biscuits-cookies' LIMIT 1)
WHERE (name ILIKE '%biscuit%' OR name ILIKE '%cookie%' OR name ILIKE '%parle%' OR name ILIKE '%biskut%')
  AND category_id IS NULL;

-- Bread & Bakery
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'bread-bakery' LIMIT 1)
WHERE (name ILIKE '%bread%' OR name ILIKE '%pav%' OR name ILIKE '%bun%' OR name ILIKE '%rusk%')
  AND category_id IS NULL;

-- Soap & Bath
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'soap-bath' LIMIT 1)
WHERE (name ILIKE '%soap%' OR name ILIKE '%dettol%' OR name ILIKE '%body wash%' OR name ILIKE '%sabun%')
  AND category_id IS NULL;

-- Laundry Care
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'laundry-care' LIMIT 1)
WHERE (name ILIKE '%surf%' OR name ILIKE '%detergent%' OR name ILIKE '%rin%' OR name ILIKE '%tide%' OR name ILIKE '%dishwash%')
  AND category_id IS NULL;

-- Packaged Foods
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'packaged-foods' LIMIT 1)
WHERE (name ILIKE '%maggi%' OR name ILIKE '%noodles%' OR name ILIKE '%pasta%')
  AND category_id IS NULL;

-- Flour & Atta
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'flour-atta' LIMIT 1)
WHERE (name ILIKE '%atta%' OR name ILIKE '%flour%' OR name ILIKE '%maida%' OR name ILIKE '%besan%')
  AND category_id IS NULL;

-- Rice & Grains
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'rice-grains' LIMIT 1)
WHERE (name ILIKE '%rice%' OR name ILIKE '%chawal%' OR name ILIKE '%basmati%' OR name ILIKE '%poha%')
  AND category_id IS NULL;

-- Pulses & Lentils
UPDATE products
SET category_id = (SELECT id FROM categories WHERE slug = 'pulses-lentils' LIMIT 1)
WHERE (name ILIKE '%dal%' OR name ILIKE '%daal%' OR name ILIKE '%chana%' OR name ILIKE '%rajma%')
  AND category_id IS NULL;
