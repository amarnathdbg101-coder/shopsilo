package utils

import (
	"strings"
	"unicode"
)

// hinglishDictionary maps everyday Hindi/Hinglish colloquial retail terms
// to both English and phonetic variations so village and rural customers
// can find products without needing exact English dictionary spelling.
var hinglishDictionary = map[string][]string{
	// Dairy, Milk & Breakfast
	"doodh":      {"doodh", "milk", "dudh", "dairy"},
	"dudh":       {"milk", "doodh", "dairy"},
	"dhoodh":     {"milk", "doodh"},
	"milk":       {"milk", "doodh", "dudh", "dairy"},
	"dahi":       {"curd", "dahi", "yogurt"},
	"curd":       {"curd", "dahi"},
	"paneer":     {"paneer", "cheese", "panir"},
	"panir":      {"paneer", "cheese"},
	"cheese":     {"cheese", "paneer"},
	"makhan":     {"butter", "makhan", "makkhan"},
	"makkhan":    {"butter", "makhan"},
	"butter":     {"butter", "makhan"},
	"malai":      {"cream", "malai"},
	"cream":      {"cream", "malai"},
	"bread":      {"bread", "pav", "bun", "bakery"},
	"pav":        {"bread", "pav", "bun"},
	"bun":        {"bun", "bread"},
	"anda":       {"egg", "anda", "ande"},
	"ande":       {"egg", "anda", "ande"},
	"egg":        {"egg", "anda", "ande"},
	"eggs":       {"egg", "anda", "ande"},

	// Cooking Oils, Ghee & Mustard
	"tel":        {"oil", "tel", "mustard", "sarson", "refined", "sunflower"},
	"tail":       {"oil", "tel", "mustard"},
	"oil":        {"oil", "tel", "mustard", "sunflower", "refined", "sarson"},
	"sarson":     {"mustard", "sarson", "oil", "tel"},
	"sarso":      {"mustard", "sarson", "oil", "tel"},
	"mustard":    {"mustard", "sarson", "oil", "tel"},
	"ghee":       {"ghee", "ghi"},
	"ghi":        {"ghee", "ghi"},

	// Grains, Rice, Atta & Pulses
	"chawal":     {"rice", "chawal", "chaawal", "basmati"},
	"chaawal":    {"rice", "chawal", "basmati"},
	"rice":       {"rice", "chawal", "chaawal", "basmati"},
	"basmati":    {"basmati", "rice", "chawal"},
	"atta":       {"atta", "flour", "gehu", "wheat", "chakki", "aata"},
	"aata":       {"atta", "flour", "gehu", "wheat", "chakki"},
	"gehu":       {"wheat", "atta", "flour", "gehu"},
	"wheat":      {"wheat", "atta", "gehu"},
	"flour":      {"flour", "atta", "maida", "besan"},
	"maida":      {"maida", "flour"},
	"besan":      {"besan", "gram flour", "chana"},
	"suji":       {"suji", "sooji", "semolina"},
	"sooji":      {"sooji", "suji", "semolina"},
	"poha":       {"poha", "chura", "flattened rice"},
	"chura":      {"poha", "chura", "rice"},
	"dal":        {"dal", "daal", "pulse", "lentil", "arhar", "masoor", "moong", "chana", "rajma", "urad"},
	"daal":       {"dal", "daal", "pulse", "lentil", "arhar", "masoor", "moong", "chana", "rajma", "urad"},
	"pulse":      {"dal", "daal", "pulse", "lentil"},
	"lentil":     {"dal", "daal", "pulse", "lentil"},
	"arhar":      {"arhar", "toor", "dal"},
	"toor":       {"toor", "arhar", "dal"},
	"masoor":     {"masoor", "dal"},
	"moong":      {"moong", "mung", "dal"},
	"chana":      {"chana", "gram", "chhole", "dal"},
	"chhole":     {"chhole", "chana", "gram"},
	"rajma":      {"rajma", "kidney beans"},
	"urad":       {"urad", "dal"},

	// Sweet, Salt & Spices
	"cheeni":     {"sugar", "cheeni", "chini", "shakkar", "sweetener"},
	"chini":      {"sugar", "cheeni", "chini", "shakkar"},
	"shakkar":    {"sugar", "cheeni", "shakkar"},
	"sakkar":     {"sugar", "cheeni", "shakkar"},
	"sugar":      {"sugar", "cheeni", "chini"},
	"gud":        {"jaggery", "gud", "gur"},
	"gur":        {"jaggery", "gud", "gur"},
	"namak":      {"salt", "namak", "iodized"},
	"salt":       {"salt", "namak", "iodized"},
	"masala":     {"masala", "spice", "powder"},
	"masale":     {"masala", "spice", "powder"},
	"spices":     {"masala", "spice"},
	"mirch":      {"chilli", "chili", "mirch", "mirchi"},
	"mirchi":     {"chilli", "chili", "mirch", "mirchi"},
	"chilli":     {"chilli", "chili", "mirch", "mirchi"},
	"haldi":      {"turmeric", "haldi"},
	"turmeric":   {"turmeric", "haldi"},
	"dhaniya":    {"coriander", "dhaniya", "dhania"},
	"dhania":     {"coriander", "dhaniya"},
	"coriander":  {"coriander", "dhaniya"},
	"jeera":      {"jeera", "zeera", "cumin"},
	"zeera":      {"jeera", "zeera", "cumin"},
	"cumin":      {"jeera", "zeera", "cumin"},
	"adrak":      {"ginger", "adrak"},
	"ginger":     {"ginger", "adrak"},
	"lehsun":     {"garlic", "lehsun", "lahsun"},
	"lahsun":     {"garlic", "lehsun", "lahsun"},
	"garlic":     {"garlic", "lehsun", "lahsun"},
	"elaichi":    {"elaichi", "cardamom"},
	"cardamom":   {"elaichi", "cardamom"},
	"laung":      {"laung", "clove"},
	"clove":      {"laung", "clove"},
	"hing":       {"hing", "asafoetida"},
	"rai":        {"mustard seed", "rai", "sarson"},

	// Tea, Coffee & Beverages
	"chai":       {"tea", "chai", "chay", "chaipatti"},
	"chay":       {"tea", "chai", "chaipatti"},
	"chaipatti":  {"tea", "chai", "chaipatti"},
	"tea":        {"tea", "chai", "chaipatti"},
	"coffee":     {"coffee", "kafi"},
	"kafi":       {"coffee"},
	"paani":      {"water", "paani", "pani", "soda", "drink"},
	"pani":       {"water", "paani", "pani", "soda", "drink"},
	"water":      {"water", "paani", "pani", "bisleri"},
	"cold drink": {"drink", "beverage", "soda", "coke", "pepsi", "sprite", "thums"},
	"sharbat":    {"sharbat", "syrup", "squash", "juice"},
	"juice":      {"juice", "drink", "beverage"},

	// Biscuits, Snacks & Sweets
	"biscuit":    {"biscuit", "biskut", "cookie", "cookies", "rusk"},
	"biskut":     {"biscuit", "biskut", "cookie"},
	"cookies":    {"biscuit", "cookie", "cookies"},
	"rusk":       {"rusk", "toast", "biscuit"},
	"toast":      {"rusk", "toast"},
	"namkeen":    {"namkeen", "bhujia", "sev", "mixture", "snack"},
	"bhujia":     {"bhujia", "namkeen", "sev"},
	"sev":        {"sev", "bhujia", "namkeen"},
	"chips":      {"chips", "wafer", "snack", "kurkure", "lays"},
	"kurkure":    {"kurkure", "chips", "namkeen", "snack"},
	"chocolate":  {"chocolate", "choclate", "cadbury", "sweet", "dairy milk"},
	"choclate":   {"chocolate", "cadbury"},
	"mithai":     {"sweet", "mithai", "meetha"},

	// Personal Care, Soaps & Hygiene
	"sabun":      {"soap", "sabun", "saabun", "bar", "bath"},
	"saabun":     {"soap", "sabun", "bar"},
	"soap":       {"soap", "sabun", "bar", "bath"},
	"shampoo":    {"shampoo", "shampu", "hair"},
	"shampu":     {"shampoo", "hair"},
	"tel baal":   {"hair oil", "kesh", "tel", "amla", "parachute", "coconut"},
	"manjan":     {"toothpaste", "paste", "colgate", "oral", "manjan", "brush"},
	"paste":      {"toothpaste", "paste", "colgate", "oral"},
	"colgate":    {"colgate", "toothpaste", "paste"},
	"toothpaste": {"toothpaste", "paste", "colgate", "oral", "manjan"},
	"brush":      {"toothbrush", "brush"},

	// Cleaning & Laundry
	"surf":       {"detergent", "surf", "washing powder", "laundry", "wash", "ghadi", "tide"},
	"detergent":  {"detergent", "surf", "washing powder", "wash", "powder", "ghadi", "tide"},
	"nirma":      {"detergent", "nirma", "washing powder", "surf"},
	"ghadi":      {"ghadi", "detergent", "soap"},
	"tide":       {"tide", "detergent"},
	"rin":        {"rin", "detergent", "soap"},
	"harpic":     {"harpic", "cleaner", "toilet"},
	"cleaner":    {"cleaner", "harpic", "lizol", "surface", "floor"},
	"jhadu":      {"broom", "jhadu", "cleaning"},
	"pocha":      {"wiper", "mop", "pocha"},

	// Health & Medicine
	"dawa":       {"medicine", "dawa", "davai", "tablet", "syrup", "pharma"},
	"davai":      {"medicine", "dawa", "davai", "tablet"},
	"dawaii":     {"medicine", "dawa", "davai"},
	"medicine":   {"medicine", "dawa", "davai", "tablet", "pharma"},
	"tablet":     {"tablet", "capsule", "medicine", "goli"},
	"goli":       {"tablet", "medicine", "goli"},
	"bandage":    {"bandage", "plaster", "first aid", "dettol"},
	"dettol":     {"dettol", "antiseptic", "soap"},

	// Vegetables & Fruits
	"aloo":       {"potato", "aloo", "aalu"},
	"aalu":       {"potato", "aloo"},
	"potato":     {"potato", "aloo", "aalu"},
	"pyaz":       {"onion", "pyaz", "pyaaz"},
	"pyaaz":      {"onion", "pyaz"},
	"onion":      {"onion", "pyaz"},
	"tamatar":    {"tomato", "tamatar"},
	"tomato":     {"tomato", "tamatar"},
	"nimbu":      {"lemon", "nimbu"},
	"lemon":      {"lemon", "nimbu"},
	"kela":       {"banana", "kela"},
	"banana":     {"banana", "kela"},
	"seb":        {"apple", "seb"},
	"apple":      {"apple", "seb"},
	"aam":        {"mango", "aam"},
	"mango":      {"mango", "aam"},

	// Clothing & Footwear
	"kapda":      {"clothes", "clothing", "shirt", "pant", "kurta", "kapda"},
	"kapde":      {"clothes", "clothing", "shirt", "pant", "kapda"},
	"clothes":    {"clothes", "clothing", "kapda", "shirt"},
	"clothing":   {"clothing", "clothes", "kapda"},
	"shirt":      {"shirt", "tshirt", "clothing"},
	"pant":       {"pant", "jeans", "trouser"},
	"kurta":      {"kurta", "kurti", "clothing"},
	"saree":      {"saree", "sari", "clothing"},
	"sari":       {"saree", "sari"},
	"joota":      {"shoes", "slipper", "chappal", "sandals", "footwear", "juta"},
	"juta":       {"shoes", "joota", "chappal"},
	"chappal":    {"slipper", "chappal", "sandals", "flipflop"},
	"shoes":      {"shoes", "joota", "sneakers", "footwear"},
	"slipper":    {"slipper", "chappal", "flipflop"},
}

// CleanSearchQuery removes punctuation and normalizes whitespace
func CleanSearchQuery(q string) string {
	var sb strings.Builder
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// SearchTermGroup represents a group of synonyms for a given search word.
// In SQL: (field LIKE '%term1%' OR field LIKE '%term2%')
type SearchTermGroup struct {
	Terms []string
}

// ExpandHinglishSearchGroups breaks a multi-word or single-word search query into
// phonetic/bilingual synonym groups.
// For example:
// "amul doodh" -> Group 1: ["amul"], Group 2: ["doodh", "milk", "dudh", "dairy"]
// "fortune tel" -> Group 1: ["fortune"], Group 2: ["tel", "oil", "mustard", "sarson"]
// "cheeni" -> Group 1: ["cheeni", "sugar", "chini", "shakkar"]
func ExpandHinglishSearchGroups(rawQuery string) []SearchTermGroup {
	cleaned := CleanSearchQuery(rawQuery)
	if cleaned == "" {
		return nil
	}

	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return nil
	}

	// 1. Check if the entire multi-word phrase matches a single dictionary key (e.g. "cold drink", "tel baal")
	if syns, ok := hinglishDictionary[cleaned]; ok {
		return []SearchTermGroup{{Terms: syns}}
	}

	var groups []SearchTermGroup
	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		// Look up word in dictionary
		if syns, ok := hinglishDictionary[word]; ok {
			groups = append(groups, SearchTermGroup{Terms: syns})
		} else {
			// Plain word fallback (matches word itself)
			groups = append(groups, SearchTermGroup{Terms: []string{word}})
		}
	}

	return groups
}
