package services

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"shopMe/internal/handler/dto"
)

// Retail unit normalization mapping (Hindi, Hinglish and English Kirana units)
var retailUnits = map[string]string{
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
}

// Conversational words that suggest open-ended Hinglish intent requiring Gemini LLM reasoning
var conversationalStopWords = []string{
	"bhaiya", "bhaya", "suno", "chahiye", "dena", "dedo", "kripya", "please",
	"hoga", "dikhao", "batao", "kaun", "kya", "discount", "offer", "sasta",
	"recommend", "achha", "kharidna", "shandar", "namaste", "hello",
}

// Regex patterns for local item parsing
var (
	// Pattern 1: Price first (e.g., "50 ka tel", "100 rupaye ki cheeni", "40 rs dahi")
	patternPriceFirst = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(?:ka|ki|ke|rs|rupaye|inr|₹)\s*(?:ka|ki|ke)?\s*(.+)$`)

	// Pattern 2: Qty + Unit + Name + optional price (e.g., "2 kg cheeni", "1 packet amul butter", "3 litre tel 150 rs")
	patternQtyUnitName = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(kg|kilo|kilogram|gm|gram|g|grams|packet|pkt|pouch|packets|litre|liter|ltr|litres|l|lt|ml|pcs|pc|piece|pieces|peice|botal|bottle|bottles|dabba|box|boxes)?\s*(?:ka|ki|ke)?\s*(.+?)(?:\s+(?:rs|rupaye|inr|₹|ka|ki)?\s*(\d+(?:\.\d+)?)\s*(?:rs|rupaye|inr|₹)?)?$`)

	// Pattern 3: Name + Qty + Unit + optional price (e.g., "cheeni 2 kg", "amul butter 1 packet 55 rs", "dettol soap 2 pcs")
	patternNameQtyUnit = regexp.MustCompile(`(?i)^(.+?)\s+(\d+(?:\.\d+)?)\s*(kg|kilo|kilogram|gm|gram|g|grams|packet|pkt|pouch|packets|litre|liter|ltr|litres|l|lt|ml|pcs|pc|piece|pieces|peice|botal|bottle|bottles|dabba|box|boxes)(?:\s+(?:rs|rupaye|inr|₹|ka|ki)?\s*(\d+(?:\.\d+)?)\s*(?:rs|rupaye|inr|₹)?)?$`)

	// Bullets / numbers prefix on grocery slips (e.g., "1. ", "2) ", "- ", "* ")
	bulletPrefix = regexp.MustCompile(`^\s*(?:\d+[\.\)]|[-*•])\s*`)
)

// cleanTitle converts raw strings to readable Title Case (e.g. "amul butter" -> "Amul Butter")
func cleanTitle(s string) string {
	words := strings.Fields(strings.TrimSpace(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// normalizeRetailUnit maps Hindi and English unit synonyms to standard retail units
func normalizeRetailUnit(raw string) string {
	clean := strings.ToLower(strings.TrimSpace(raw))
	if u, ok := retailUnits[clean]; ok {
		return u
	}
	if clean == "" {
		return "pcs"
	}
	return clean
}

// hasConversationalStopWords checks if the input contains natural speech markers requiring LLM reasoning
func hasConversationalStopWords(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range conversationalStopWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// parseSingleItem parses a single line/phrase into a structured VoiceBillItem
func parseSingleItem(line string) (*dto.VoiceBillItem, bool) {
	clean := strings.TrimSpace(line)
	if clean == "" {
		return nil, false
	}

	// If line has conversational stop words, let Gemini handle it
	if hasConversationalStopWords(clean) {
		return nil, false
	}

	// 1. Check Pattern 1: Price first (e.g. "50 ka tel")
	if m := patternPriceFirst.FindStringSubmatch(clean); len(m) == 3 {
		price, _ := strconv.ParseFloat(m[1], 64)
		name := strings.TrimSpace(m[2])
		if name != "" && len(name) > 1 {
			return &dto.VoiceBillItem{
				ProductName: cleanTitle(name),
				Quantity:    1.0,
				Unit:        "pcs",
				UnitPrice:   price,
				TotalPrice:  price,
			}, true
		}
	}

	// 2. Check Pattern 3: Name + Qty + Unit (e.g. "cheeni 2 kg 80 rs")
	if m := patternNameQtyUnit.FindStringSubmatch(clean); len(m) >= 4 {
		name := strings.TrimSpace(m[1])
		qty, _ := strconv.ParseFloat(m[2], 64)
		unit := normalizeRetailUnit(m[3])
		var price float64
		if len(m) >= 5 && m[4] != "" {
			price, _ = strconv.ParseFloat(m[4], 64)
		}
		if name != "" && qty > 0 {
			totalPrice := price
			unitPrice := price
			if price > 0 && qty > 0 {
				unitPrice = math.Round((price/qty)*100) / 100
			}
			return &dto.VoiceBillItem{
				ProductName: cleanTitle(name),
				Quantity:    qty,
				Unit:        unit,
				UnitPrice:   unitPrice,
				TotalPrice:  totalPrice,
			}, true
		}
	}

	// 3. Check Pattern 2: Qty + Unit + Name (e.g. "2 kg cheeni", "1 amul butter")
	if m := patternQtyUnitName.FindStringSubmatch(clean); len(m) >= 4 {
		qty, _ := strconv.ParseFloat(m[1], 64)
		unit := normalizeRetailUnit(m[2])
		name := strings.TrimSpace(m[3])
		var price float64
		if len(m) >= 5 && m[4] != "" {
			price, _ = strconv.ParseFloat(m[4], 64)
		}
		if name != "" && qty > 0 {
			totalPrice := price
			unitPrice := price
			if price > 0 && qty > 0 {
				unitPrice = math.Round((price/qty)*100) / 100
			}
			return &dto.VoiceBillItem{
				ProductName: cleanTitle(name),
				Quantity:    qty,
				Unit:        unit,
				UnitPrice:   unitPrice,
				TotalPrice:  totalPrice,
			}, true
		}
	}

	return nil, false
}

// splitVoiceCommands splits spoken text by common separators: comma, newline, "aur", "and", "+"
func splitVoiceCommands(input string) []string {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "+", ",")

	// Replace " aur " and " and " with commas
	reAnd := regexp.MustCompile(`(?i)\s+(?:aur|and)\s+`)
	normalized = reAnd.ReplaceAllString(normalized, ",")

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || r == '\n'
	})

	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseVoiceBillLocally evaluates counter billing voice commands locally using fast regex heuristics.
// Returns parsed response and true if all spoken items were matched with 100% confidence, saving AI tokens.
func parseVoiceBillLocally(spokenText string) (*dto.VoiceBillResponse, bool) {
	clean := strings.TrimSpace(spokenText)
	if clean == "" {
		return nil, false
	}

	segments := splitVoiceCommands(clean)
	if len(segments) == 0 {
		return nil, false
	}

	var matched []dto.VoiceBillItem
	var total float64

	for _, seg := range segments {
		item, ok := parseSingleItem(seg)
		if !ok {
			// Any segment failed to resolve locally; fallback to Gemini AI for complete context
			return nil, false
		}
		matched = append(matched, *item)
		total += item.TotalPrice
	}

	return &dto.VoiceBillResponse{
		SpokenText:    spokenText,
		MatchedItems:  matched,
		UnmatchedText: []string{},
		TotalAmount:   math.Round(total*100) / 100,
	}, true
}

// parseParchiLocally parses multiline grocery list slips locally using fast regex heuristics.
// If all grocery lines can be resolved locally without ambiguity, returns ParseParchiResponse and true.
func parseParchiLocally(rawText string) (*dto.ParseParchiResponse, bool) {
	lines := strings.Split(strings.ReplaceAll(rawText, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, false
	}

	var matched []*dto.ParsedParchiItem
	var unmatched []*dto.UnmatchedParchiLine
	totalLines := 0
	var total float64

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		totalLines++
		// Strip leading bullet numbers like "1. ", "2) ", "- "
		cleanedLine := bulletPrefix.ReplaceAllString(trimmed, "")
		cleanedLine = strings.TrimSpace(cleanedLine)

		item, ok := parseSingleItem(cleanedLine)
		if !ok {
			unmatched = append(unmatched, &dto.UnmatchedParchiLine{
				OriginalLine: trimmed,
				QueryTerm:    cleanedLine,
				Reason:       "Could not determine quantity and product name",
			})
			continue
		}

		qtyInt := int(math.Ceil(item.Quantity))
		if qtyInt < 1 {
			qtyInt = 1
		}

		matched = append(matched, &dto.ParsedParchiItem{
			ProductName:       item.ProductName,
			RequestedQuantity: qtyInt,
			ParsedUnit:        item.Unit,
			UnitPrice:         item.UnitPrice,
			TotalPrice:        item.TotalPrice,
			AvailableStock:    99,
			InStock:           true,
		})
		total += item.TotalPrice
	}

	// If there are zero matched items or any unmatched lines, let Gemini AI handle it
	if len(matched) == 0 || len(unmatched) > 0 {
		return nil, false
	}

	return &dto.ParseParchiResponse{
		TotalLinesParsed:     totalLines,
		MatchedCount:         len(matched),
		UnmatchedCount:       0,
		EstimatedTotalAmount: math.Round(total*100) / 100,
		MatchedItems:         matched,
		UnmatchedLines:       []*dto.UnmatchedParchiLine{},
	}, true
}
