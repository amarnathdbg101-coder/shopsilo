package services

import (
	"testing"
)

func TestParseVoiceBillLocally(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectSuccess bool
		expectedItems int
		expectedTotal float64
	}{
		{
			name:          "Single Item Standard Unit",
			input:         "2 kg cheeni",
			expectSuccess: true,
			expectedItems: 1,
			expectedTotal: 0,
		},
		{
			name:          "Single Item With Price",
			input:         "2 kg cheeni 80 rs",
			expectSuccess: true,
			expectedItems: 1,
			expectedTotal: 80,
		},
		{
			name:          "Price First (Hindi phrasing)",
			input:         "50 ka tel",
			expectSuccess: true,
			expectedItems: 1,
			expectedTotal: 50,
		},
		{
			name:          "Compound with 'aur' separator",
			input:         "2 kg cheeni aur 1 packet amul butter",
			expectSuccess: true,
			expectedItems: 2,
			expectedTotal: 0,
		},
		{
			name:          "Comma separated multi-item with prices",
			input:         "1 bread 40 rs, 2 packet maggi 28 rs",
			expectSuccess: true,
			expectedItems: 2,
			expectedTotal: 68,
		},
		{
			name:          "Name first then quantity",
			input:         "dettol soap 3 piece",
			expectSuccess: true,
			expectedItems: 1,
			expectedTotal: 0,
		},
		{
			name:          "Conversational query with stop word (should fallback to AI)",
			input:         "bhaiya 2 packet doodh dena jo fresh ho",
			expectSuccess: false,
		},
		{
			name:          "Open-ended recommendation question (should fallback to AI)",
			input:         "kuch sasta discount offer batao",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, ok := parseVoiceBillLocally(tt.input)
			if ok != tt.expectSuccess {
				t.Fatalf("expected success=%v, got %v for input %q", tt.expectSuccess, ok, tt.input)
			}
			if ok {
				if len(resp.MatchedItems) != tt.expectedItems {
					t.Errorf("expected %d items, got %d", tt.expectedItems, len(resp.MatchedItems))
				}
				if tt.expectedTotal > 0 && resp.TotalAmount != tt.expectedTotal {
					t.Errorf("expected total %.2f, got %.2f", tt.expectedTotal, resp.TotalAmount)
				}
			}
		})
	}
}

func TestParseParchiLocally(t *testing.T) {
	parchiText := `1. 2 kg cheeni 80 rs
2. 1 packet namak 28 rs
3. 500 gm rajma 60 rs`

	resp, ok := parseParchiLocally(parchiText)
	if !ok {
		t.Fatalf("expected parchi to parse locally, got false")
	}

	if resp.MatchedCount != 3 {
		t.Errorf("expected 3 matched items, got %d", resp.MatchedCount)
	}

	if resp.EstimatedTotalAmount != 168.0 {
		t.Errorf("expected total amount 168.0, got %.2f", resp.EstimatedTotalAmount)
	}

	// Test unparseable/conversational text falls back to AI
	messyParchi := `1. 2 kg cheeni
bhaiya thoda sa dhaniya bhi dal dena
aur koi accha biscuit`

	_, messyOk := parseParchiLocally(messyParchi)
	if messyOk {
		t.Errorf("expected messy parchi to fallback to AI (return false), but got true")
	}
}
