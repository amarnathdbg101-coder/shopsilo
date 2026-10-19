package utils

import (
	"regexp"
	"strings"
)

// CanonicalCategoryInfo defines standard metadata and synonyms for a canonical category.
type CanonicalCategoryInfo struct {
	Slug         string
	NameEn       string
	NameHi       string
	Department   string
	Icon         string
	SynonymSlugs []string
	SynonymNames []string
}

// CanonicalCategories holds the 21 standard retail categories
var CanonicalCategories = []CanonicalCategoryInfo{
	{
		Slug:       "kirana-grocery",
		NameEn:     "Kirana & Grocery",
		NameHi:     "किराना व राशन",
		Department: "grocery",
		Icon:       "🛒",
		SynonymSlugs: []string{
			"kirana-grocery", "kirana-and-grocery", "grocery-and-staples", "grocery-staples",
			"groceries-and-staples", "groceries-staples", "grocery", "groceries", "staples",
			"kirana", "dal-pulses", "pulses-lentils", "rice-grains", "flour-atta", "edible-oils",
			"sugar-salt", "spices-masala", "packaged-foods",
		},
		SynonymNames: []string{
			"Kirana & Grocery", "Grocery & Staples", "Groceries & Staples",
			"Kirana and Grocery", "Grocery and Staples", "Groceries and Staples",
		},
	},
	{
		Slug:       "dairy-breakfast",
		NameEn:     "Dairy & Breakfast",
		NameHi:     "डेयरी व नाश्ता",
		Department: "dairy",
		Icon:       "🥛",
		SynonymSlugs: []string{
			"dairy-breakfast", "dairy-and-breakfast", "dairy-bakery-and-frozen", "dairy-bakery-frozen",
			"dairy-and-bakery", "dairy-bakery", "dairy-bread-and-eggs", "dairy-bread-eggs",
			"dairy", "breakfast", "milk-curd", "butter-cheese", "bread-bakery", "eggs",
		},
		SynonymNames: []string{
			"Dairy & Breakfast", "Dairy, Bakery & Frozen", "Dairy and Breakfast",
			"Dairy & Bakery", "Dairy and Bakery", "Dairy, Bread & Eggs",
		},
	},
	{
		Slug:       "snacks-beverages",
		NameEn:     "Snacks & Beverages",
		NameHi:     "स्नैक्स व पेय",
		Department: "snacks",
		Icon:       "🍿",
		SynonymSlugs: []string{
			"snacks-beverages", "snacks-and-beverages", "snacks-and-munchies", "snacks-munchies",
			"beverages-and-drinks", "beverages-drinks", "beverages", "drinks", "snacks",
			"munchies", "namkeen", "chips-namkeen", "biscuits-cookies", "cold-drinks",
			"tea-coffee", "soft-drinks-juices",
		},
		SynonymNames: []string{
			"Snacks & Beverages", "Snacks & Munchies", "Beverages & Drinks",
			"Snacks and Beverages", "Snacks and Munchies", "Beverages and Drinks",
		},
	},
	{
		Slug:       "bakery-cakes",
		NameEn:     "Bakery & Cakes",
		NameHi:     "बेकरी व केक",
		Department: "dairy",
		Icon:       "🎂",
		SynonymSlugs: []string{
			"bakery-cakes", "bakery-and-cakes", "bakery", "cakes", "pastry", "bread-bakery", "rusk",
		},
		SynonymNames: []string{
			"Bakery & Cakes", "Bakery and Cakes", "Bakery",
		},
	},
	{
		Slug:       "fruits-vegetables",
		NameEn:     "Fruits & Vegetables",
		NameHi:     "फल व सब्जियां",
		Department: "grocery",
		Icon:       "🍎",
		SynonymSlugs: []string{
			"fruits-vegetables", "fruits-and-vegetables", "fruits", "vegetables",
			"fresh-fruits", "fresh-vegetables", "sabji",
		},
		SynonymNames: []string{
			"Fruits & Vegetables", "Fruits and Vegetables",
		},
	},
	{
		Slug:       "personal-care",
		NameEn:     "Personal Care",
		NameHi:     "पर्सनल केयर",
		Department: "personal-care",
		Icon:       "🧴",
		SynonymSlugs: []string{
			"personal-care", "personal-care-and-beauty", "personal-care-beauty",
			"hair-care", "oral-care", "skin-care", "soap-bath", "bath-body",
		},
		SynonymNames: []string{
			"Personal Care", "Personal Care & Beauty", "Personal Care and Beauty",
		},
	},
	{
		Slug:       "beauty-cosmetics",
		NameEn:     "Beauty & Cosmetics",
		NameHi:     "सौंदर्य व मेकअप",
		Department: "personal-care",
		Icon:       "💄",
		SynonymSlugs: []string{
			"beauty-cosmetics", "beauty-and-cosmetics", "beauty", "cosmetics", "makeup",
		},
		SynonymNames: []string{
			"Beauty & Cosmetics", "Beauty and Cosmetics",
		},
	},
	{
		Slug:       "household-cleaning",
		NameEn:     "Household Cleaning",
		NameHi:     "घर व साफ-सफाई",
		Department: "home",
		Icon:       "🧹",
		SynonymSlugs: []string{
			"household-cleaning", "household-and-cleaning", "home-cleaning",
			"laundry-care", "detergents", "cleaning", "household",
		},
		SynonymNames: []string{
			"Household Cleaning", "Household & Cleaning", "Household and Cleaning",
		},
	},
	{
		Slug:       "home-kitchen",
		NameEn:     "Home & Kitchen",
		NameHi:     "रसोई व घरेलू सामान",
		Department: "home",
		Icon:       "🍳",
		SynonymSlugs: []string{
			"home-kitchen", "home-and-kitchen", "kitchen-supplies", "cookware",
			"utensils", "containers", "kitchen",
		},
		SynonymNames: []string{
			"Home & Kitchen", "Home and Kitchen",
		},
	},
	{
		Slug:       "pooja-spiritual",
		NameEn:     "Pooja & Spiritual",
		NameHi:     "पूजा व धार्मिक सामग्री",
		Department: "home",
		Icon:       "🪔",
		SynonymSlugs: []string{
			"pooja-spiritual", "pooja-and-spiritual", "puja-religious", "pooja-essentials",
			"agarbatti", "camphor", "diya", "puja",
		},
		SynonymNames: []string{
			"Pooja & Spiritual", "Pooja and Spiritual", "Puja & Religious",
		},
	},
	{
		Slug:       "baby-care",
		NameEn:     "Baby Care",
		NameHi:     "शिशु देखभाल",
		Department: "personal-care",
		Icon:       "👶",
		SynonymSlugs: []string{
			"baby-care", "baby-care-and-toys", "toys-and-baby-products", "toys-baby-products",
			"baby-food", "diapers", "wipes", "baby-products", "baby",
		},
		SynonymNames: []string{
			"Baby Care", "Baby Care & Toys", "Toys & Baby Products", "Baby Care and Toys",
		},
	},
	{
		Slug:       "pharmacy-healthcare",
		NameEn:     "Pharmacy & Healthcare",
		NameHi:     "दवाई व स्वास्थ्य",
		Department: "personal-care",
		Icon:       "💊",
		SynonymSlugs: []string{
			"pharmacy-healthcare", "pharmacy-and-healthcare", "medicine-and-healthcare",
			"medicine-healthcare", "pharmacy", "medicine", "healthcare",
			"first-aid", "vitamins-supplements", "otc",
		},
		SynonymNames: []string{
			"Pharmacy & Healthcare", "Medicine & Healthcare", "Pharmacy and Healthcare", "Medicine and Healthcare",
		},
	},
	{
		Slug:       "electronics-accessories",
		NameEn:     "Electronics & Accessories",
		NameHi:     "इलेक्ट्रॉनिक्स व उपकरण",
		Department: "electronics",
		Icon:       "⚡",
		SynonymSlugs: []string{
			"electronics-accessories", "electronics-and-accessories", "electronics-and-appliances",
			"electronics-appliances", "electronics-and-mobile", "mobile-and-accessories",
			"mobile-accessories", "electronics", "appliances", "gadgets", "mobile",
		},
		SynonymNames: []string{
			"Electronics & Accessories", "Electronics & Appliances", "Mobile & Accessories",
			"Electronics and Appliances", "Electronics and Mobile",
		},
	},
	{
		Slug:       "clothing-apparel",
		NameEn:     "Clothing & Apparel",
		NameHi:     "कपड़े व परिधान",
		Department: "fashion",
		Icon:       "👕",
		SynonymSlugs: []string{
			"clothing-apparel", "clothing-and-apparel", "clothing-and-fashion",
			"clothing-fashion", "mens-clothing", "womens-clothing", "kids-clothing",
			"clothing", "apparel", "fashion",
		},
		SynonymNames: []string{
			"Clothing & Apparel", "Clothing & Fashion", "Clothing and Apparel", "Clothing and Fashion",
		},
	},
	{
		Slug:       "footwear",
		NameEn:     "Footwear",
		NameHi:     "जूते व चप्पल",
		Department: "fashion",
		Icon:       "👟",
		SynonymSlugs: []string{
			"footwear", "footwear-and-leather", "footwear-leather", "shoes", "sandals", "slippers", "leather",
		},
		SynonymNames: []string{
			"Footwear", "Footwear & Leather", "Footwear and Leather",
		},
	},
	{
		Slug:       "stationery-office",
		NameEn:     "Stationery & Office",
		NameHi:     "स्टेशनरी व ऑफिस",
		Department: "stationery",
		Icon:       "📚",
		SynonymSlugs: []string{
			"stationery-office", "stationery-and-office", "stationery-and-office-supplies",
			"stationery-office-supplies", "notebooks", "books-magazines", "pens", "stationery", "office",
		},
		SynonymNames: []string{
			"Stationery & Office", "Stationery & Office Supplies", "Stationery and Office Supplies",
		},
	},
	{
		Slug:       "hardware-tools",
		NameEn:     "Hardware & Tools",
		NameHi:     "हार्डवेयर व टूल्स",
		Department: "home",
		Icon:       "🔧",
		SynonymSlugs: []string{
			"hardware-tools", "hardware-and-tools", "hardware-and-building-material",
			"hardware-building-material", "hardware-electrical", "locks", "tools", "hardware",
			"automotive-and-spare-parts", "automotive-spare-parts", "automotive",
		},
		SynonymNames: []string{
			"Hardware & Tools", "Hardware & Building Material", "Hardware and Tools", "Hardware and Building Material",
		},
	},
	{
		Slug:       "sports-fitness",
		NameEn:     "Sports & Fitness",
		NameHi:     "खेल व फिटनेस",
		Department: "sports",
		Icon:       "🏏",
		SynonymSlugs: []string{
			"sports-fitness", "sports-and-fitness", "sports", "fitness", "gym", "cricket",
		},
		SynonymNames: []string{
			"Sports & Fitness", "Sports and Fitness",
		},
	},
	{
		Slug:       "toys-games",
		NameEn:     "Toys & Games",
		NameHi:     "खिलौने व गेम्स",
		Department: "toys",
		Icon:       "🧸",
		SynonymSlugs: []string{
			"toys-games", "toys-and-games", "toys", "games", "board-games", "puzzles",
		},
		SynonymNames: []string{
			"Toys & Games", "Toys and Games",
		},
	},
	{
		Slug:       "pet-care",
		NameEn:     "Pet Care",
		NameHi:     "पालतू सामान",
		Department: "home",
		Icon:       "🐾",
		SynonymSlugs: []string{
			"pet-care", "pet-supplies", "pet-care-and-supplies", "pets", "dog-food", "cat-food",
		},
		SynonymNames: []string{
			"Pet Care", "Pet Care & Supplies", "Pet Supplies", "Pet Care and Supplies",
		},
	},
	{
		Slug:       "general-store",
		NameEn:     "General Store",
		NameHi:     "जनरल स्टोर",
		Department: "all",
		Icon:       "🏪",
		SynonymSlugs: []string{
			"general-store", "general", "services-and-others", "services-others", "services", "others", "other",
		},
		SynonymNames: []string{
			"General Store", "Services & Others", "Services and Others",
		},
	},
}

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	slugLookupMap        map[string]string
	infoMap              map[string]CanonicalCategoryInfo
)

func init() {
	slugLookupMap = make(map[string]string)
	infoMap = make(map[string]CanonicalCategoryInfo)

	for _, cat := range CanonicalCategories {
		infoMap[cat.Slug] = cat
		cleanCanonical := CleanCategorySlug(cat.Slug)
		slugLookupMap[cleanCanonical] = cat.Slug
		slugLookupMap[CleanCategorySlug(cat.NameEn)] = cat.Slug

		for _, syn := range cat.SynonymSlugs {
			slugLookupMap[CleanCategorySlug(syn)] = cat.Slug
		}
		for _, name := range cat.SynonymNames {
			slugLookupMap[CleanCategorySlug(name)] = cat.Slug
		}
	}
}

// CleanCategorySlug strips non-alphanumeric characters for clean fast matching.
func CleanCategorySlug(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	lower = strings.ReplaceAll(lower, "&", "and")
	cleaned := nonAlphanumericRegex.ReplaceAllString(lower, "-")
	return strings.Trim(cleaned, "-")
}

// ResolveCanonicalCategorySlug maps any category name, slug, or synonym into its canonical category slug.
func ResolveCanonicalCategorySlug(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	clean := CleanCategorySlug(trimmed)
	if canon, ok := slugLookupMap[clean]; ok {
		return canon, true
	}
	noHyphens := strings.ReplaceAll(clean, "-", "")
	for k, v := range slugLookupMap {
		if strings.ReplaceAll(k, "-", "") == noHyphens {
			return v, true
		}
	}
	return "", false
}

// GetCategorySynonymSlugs returns all slug aliases for a given canonical or synonym slug.
func GetCategorySynonymSlugs(raw string) []string {
	canonSlug, ok := ResolveCanonicalCategorySlug(raw)
	if !ok {
		canonSlug = CleanCategorySlug(raw)
	}
	if info, exists := infoMap[canonSlug]; exists {
		return info.SynonymSlugs
	}
	return []string{canonSlug}
}

// GetCategorySynonymNames returns all name aliases for a given canonical or synonym slug.
func GetCategorySynonymNames(raw string) []string {
	canonSlug, ok := ResolveCanonicalCategorySlug(raw)
	if !ok {
		return []string{raw}
	}
	if info, exists := infoMap[canonSlug]; exists {
		return info.SynonymNames
	}
	return []string{raw}
}
