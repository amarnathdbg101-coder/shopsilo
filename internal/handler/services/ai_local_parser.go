package services

import (
	"math"
	"strings"

	"github.com/amarnathdbg101-coder/go/go-parchi"
	"shopMe/internal/handler/dto"
)

// parseSingleItem parses a single line/phrase into a structured VoiceBillItem using go-parchi.
func parseSingleItem(line string) (*dto.VoiceBillItem, bool) {
	item, ok := parchi.ParseSingleLine(line)
	if !ok {
		return nil, false
	}
	return &dto.VoiceBillItem{
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Unit:        item.Unit,
		UnitPrice:   item.UnitPrice,
		TotalPrice:  item.TotalPrice,
	}, true
}

// splitVoiceCommands splits spoken text by common separators: comma, newline, "aur", "and", "+" using go-parchi.
func splitVoiceCommands(input string) []string {
	return parchi.SplitCommands(input)
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

// parseParchiLocally parses multiline grocery list slips locally using go-parchi.
// If all grocery lines can be resolved locally without ambiguity, returns ParseParchiResponse and true.
func parseParchiLocally(rawText string) (*dto.ParseParchiResponse, bool) {
	slip, ok := parchi.ParseSlip(rawText)
	if !ok || slip.MatchedCount == 0 || slip.UnmatchedCount > 0 {
		return nil, false
	}

	var matched []*dto.ParsedParchiItem
	for _, it := range slip.Items {
		qtyInt := int(math.Ceil(it.Quantity))
		if qtyInt < 1 {
			qtyInt = 1
		}
		matched = append(matched, &dto.ParsedParchiItem{
			ProductName:       it.ProductName,
			RequestedQuantity: qtyInt,
			ParsedUnit:        it.Unit,
			UnitPrice:         it.UnitPrice,
			TotalPrice:        it.TotalPrice,
			AvailableStock:    99,
			InStock:           true,
		})
	}

	return &dto.ParseParchiResponse{
		TotalLinesParsed:     slip.TotalLinesParsed,
		MatchedCount:         slip.MatchedCount,
		UnmatchedCount:       0,
		EstimatedTotalAmount: slip.EstimatedTotal,
		MatchedItems:         matched,
		UnmatchedLines:       []*dto.UnmatchedParchiLine{},
	}, true
}
