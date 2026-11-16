package parchi

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Item represents a parsed grocery or retail item line.
type Item struct {
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	RawLine     string  `json:"raw_line,omitempty"`
}

// UnmatchedLine holds an input line that could not be parsed with high certainty.
type UnmatchedLine struct {
	OriginalLine string `json:"original_line"`
	CleanedLine  string `json:"cleaned_line"`
	Reason       string `json:"reason"`
}

// Slip represents the parsed result of a multi-line grocery list slip.
type Slip struct {
	TotalLinesParsed int             `json:"total_lines_parsed"`
	MatchedCount     int             `json:"matched_count"`
	UnmatchedCount   int             `json:"unmatched_count"`
	EstimatedTotal   float64         `json:"estimated_total"`
	Items            []Item          `json:"items"`
	UnmatchedLines   []UnmatchedLine `json:"unmatched_lines"`
}

var (
	// Pattern 1: Price first (e.g. "50 ka tel", "100 rupaye ki cheeni", "40 rs dahi")
	patternPriceFirst = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(?:ka|ki|ke|rs|rupaye|inr|₹)\s*(?:ka|ki|ke)?\s*(.+)$`)

	// Pattern 2: Qty + Unit + Name + optional price (e.g. "2 kg cheeni", "1 packet amul butter", "3 litre tel 150 rs")
	patternQtyUnitName = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(kg|kilo|kilogram|gm|gram|g|grams|packet|pkt|pouch|packets|litre|liter|ltr|litres|l|lt|ml|pcs|pc|piece|pieces|peice|botal|bottle|bottles|dabba|box|boxes|peti|carton|bora|sack|dozen|darjan)?\s*(?:ka|ki|ke)?\s*(.+?)(?:\s+(?:rs|rupaye|inr|₹|ka|ki)?\s*(\d+(?:\.\d+)?)\s*(?:rs|rupaye|inr|₹)?)?$`)

	// Pattern 3: Name + Qty + Unit + optional price (e.g. "cheeni 2 kg", "amul butter 1 packet 55 rs", "dettol soap 2 pcs")
	patternNameQtyUnit = regexp.MustCompile(`(?i)^(.+?)\s+(\d+(?:\.\d+)?)\s*(kg|kilo|kilogram|gm|gram|g|grams|packet|pkt|pouch|packets|litre|liter|ltr|litres|l|lt|ml|pcs|pc|piece|pieces|peice|botal|bottle|bottles|dabba|box|boxes|peti|carton|bora|sack|dozen|darjan)(?:\s+(?:rs|rupaye|inr|₹|ka|ki)?\s*(\d+(?:\.\d+)?)\s*(?:rs|rupaye|inr|₹)?)?$`)

	// Bullets / list item prefix (e.g. "1. ", "2) ", "- ", "* ")
	bulletPrefix = regexp.MustCompile(`^\s*(?:\d+[\.\)]|[-*•])\s*`)

	// Splitter for compound natural phrases like "aur", "and", "+"
	reAndSplitter = regexp.MustCompile(`(?i)\s+(?:aur|and)\s+`)
)

// CleanTitle converts raw strings to Title Case with clean spacing.
func CleanTitle(s string) string {
	words := strings.Fields(strings.TrimSpace(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// SplitCommands splits a composite speech or text string by commas, newlines, "aur", "and", "+"
func SplitCommands(input string) []string {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "+", ",")
	normalized = reAndSplitter.ReplaceAllString(normalized, ",")

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

// ParseSingleLine parses a single grocery line or item phrase into a structured Item.
func ParseSingleLine(line string) (*Item, bool) {
	clean := strings.TrimSpace(line)
	if clean == "" {
		return nil, false
	}

	// If line has conversational stop words (e.g. "bhaiya discount milega kya"), skip local heuristic
	if HasConversationalStopWords(clean) {
		return nil, false
	}

	// 1. Check Pattern 1: Price first (e.g. "50 ka tel")
	if m := patternPriceFirst.FindStringSubmatch(clean); len(m) == 3 {
		price, _ := strconv.ParseFloat(m[1], 64)
		name := strings.TrimSpace(m[2])
		if name != "" && len(name) > 1 {
			return &Item{
				ProductName: CleanTitle(name),
				Quantity:    1.0,
				Unit:        "pcs",
				UnitPrice:   price,
				TotalPrice:  price,
				RawLine:     clean,
			}, true
		}
	}

	// 2. Check Pattern 3: Name + Qty + Unit (e.g. "cheeni 2 kg 80 rs")
	if m := patternNameQtyUnit.FindStringSubmatch(clean); len(m) >= 4 {
		name := strings.TrimSpace(m[1])
		qty, _ := strconv.ParseFloat(m[2], 64)
		unit := NormalizeUnit(m[3])
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
			return &Item{
				ProductName: CleanTitle(name),
				Quantity:    qty,
				Unit:        unit,
				UnitPrice:   unitPrice,
				TotalPrice:  totalPrice,
				RawLine:     clean,
			}, true
		}
	}

	// 3. Check Pattern 2: Qty + Unit + Name (e.g. "2 kg cheeni", "1 amul butter")
	if m := patternQtyUnitName.FindStringSubmatch(clean); len(m) >= 4 {
		qty, _ := strconv.ParseFloat(m[1], 64)
		unit := NormalizeUnit(m[2])
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
			return &Item{
				ProductName: CleanTitle(name),
				Quantity:    qty,
				Unit:        unit,
				UnitPrice:   unitPrice,
				TotalPrice:  totalPrice,
				RawLine:     clean,
			}, true
		}
	}

	return nil, false
}

// ParseSlip parses a multi-line or compound grocery list slip into a structured Slip.
// Returns the Slip and a boolean indicating whether 100% of the lines were parsed with high confidence.
func ParseSlip(rawText string) (*Slip, bool) {
	lines := strings.Split(strings.ReplaceAll(rawText, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, false
	}

	var matched []Item
	var unmatched []UnmatchedLine
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

		item, ok := ParseSingleLine(cleanedLine)
		if !ok {
			unmatched = append(unmatched, UnmatchedLine{
				OriginalLine: trimmed,
				CleanedLine:  cleanedLine,
				Reason:       "Could not determine quantity and product name",
			})
			continue
		}

		matched = append(matched, *item)
		total += item.TotalPrice
	}

	if totalLines == 0 {
		return nil, false
	}

	isFullSuccess := len(unmatched) == 0 && len(matched) > 0

	return &Slip{
		TotalLinesParsed: totalLines,
		MatchedCount:     len(matched),
		UnmatchedCount:   len(unmatched),
		EstimatedTotal:   math.Round(total*100) / 100,
		Items:            matched,
		UnmatchedLines:   unmatched,
	}, isFullSuccess
}
