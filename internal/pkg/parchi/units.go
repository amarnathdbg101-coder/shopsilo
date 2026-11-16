package parchi

import "strings"

// defaultRetailUnits maps colloquial Hindi, Hinglish, and English retail units to canonical forms.
var defaultRetailUnits = map[string]string{
	"kg":       "kg",
	"kilo":     "kg",
	"kilogram": "kg",
	"gm":       "gm",
	"g":        "gm",
	"gram":     "gm",
	"grams":    "gm",
	"pkt":      "packet",
	"packet":   "packet",
	"packets":  "packet",
	"pouch":    "packet",
	"l":        "litre",
	"lt":       "litre",
	"ltr":      "litre",
	"liter":    "litre",
	"litre":    "litre",
	"litres":   "litre",
	"ml":       "ml",
	"pcs":      "pcs",
	"pc":       "pcs",
	"piece":    "pcs",
	"pieces":   "pcs",
	"peice":    "pcs",
	"botal":    "bottle",
	"bottle":   "bottle",
	"bottles":  "bottle",
	"dabba":    "box",
	"box":      "box",
	"boxes":    "box",
	"peti":     "carton",
	"carton":   "carton",
	"bora":     "sack",
	"sack":     "sack",
	"dozen":    "dozen",
	"darjan":   "dozen",
}

// NormalizeUnit normalizes raw colloquial unit strings (e.g. "kilo" -> "kg", "pkt" -> "packet").
func NormalizeUnit(raw string) string {
	clean := strings.ToLower(strings.TrimSpace(raw))
	if u, ok := defaultRetailUnits[clean]; ok {
		return u
	}
	if clean == "" {
		return "pcs"
	}
	return clean
}

// defaultConversationalStopwords marks colloquial natural language indicators that imply open-ended intent.
var defaultConversationalStopwords = []string{
	"bhaiya", "bhaya", "suno", "chahiye", "dena", "dedo", "kripya", "please",
	"hoga", "dikhao", "batao", "kaun", "kya", "discount", "offer", "sasta",
	"recommend", "achha", "kharidna", "shandar", "namaste", "hello",
}

// HasConversationalStopWords checks if a text line contains conversational intent markers.
func HasConversationalStopWords(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range defaultConversationalStopwords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
