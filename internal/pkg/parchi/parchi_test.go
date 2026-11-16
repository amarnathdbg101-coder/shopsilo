package parchi_test

import (
	"testing"

	"shopMe/internal/pkg/parchi"
)

func TestParseSingleLine(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantOK        bool
		wantProdName  string
		wantQty       float64
		wantUnit      string
		wantUnitPrice float64
		wantTotal     float64
	}{
		{
			name:          "Price first - 50 ka tel",
			input:         "50 ka tel",
			wantOK:        true,
			wantProdName:  "Tel",
			wantQty:       1.0,
			wantUnit:      "pcs",
			wantUnitPrice: 50.0,
			wantTotal:     50.0,
		},
		{
			name:          "Price first - 100 rupaye ki cheeni",
			input:         "100 rupaye ki cheeni",
			wantOK:        true,
			wantProdName:  "Cheeni",
			wantQty:       1.0,
			wantUnit:      "pcs",
			wantUnitPrice: 100.0,
			wantTotal:     100.0,
		},
		{
			name:          "Qty Unit Name - 2 kg cheeni",
			input:         "2 kg cheeni",
			wantOK:        true,
			wantProdName:  "Cheeni",
			wantQty:       2.0,
			wantUnit:      "kg",
			wantUnitPrice: 0,
			wantTotal:     0,
		},
		{
			name:          "Qty Unit Name Price - 3 litre tel 150 rs",
			input:         "3 litre tel 150 rs",
			wantOK:        true,
			wantProdName:  "Tel",
			wantQty:       3.0,
			wantUnit:      "litre",
			wantUnitPrice: 50.0,
			wantTotal:     150.0,
		},
		{
			name:          "Name Qty Unit Price - amul butter 1 packet 55 rs",
			input:         "amul butter 1 packet 55 rs",
			wantOK:        true,
			wantProdName:  "Amul Butter",
			wantQty:       1.0,
			wantUnit:      "packet",
			wantUnitPrice: 55.0,
			wantTotal:     55.0,
		},
		{
			name:          "Conversational word skipped - bhaiya discount milega",
			input:         "bhaiya thoda discount milega kya",
			wantOK:        false,
		},
		{
			name:          "Empty line",
			input:         "   ",
			wantOK:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, ok := parchi.ParseSingleLine(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ParseSingleLine(%q) ok = %v; want %v", tt.input, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if item.ProductName != tt.wantProdName {
				t.Errorf("ProductName = %q; want %q", item.ProductName, tt.wantProdName)
			}
			if item.Quantity != tt.wantQty {
				t.Errorf("Quantity = %v; want %v", item.Quantity, tt.wantQty)
			}
			if item.Unit != tt.wantUnit {
				t.Errorf("Unit = %q; want %q", item.Unit, tt.wantUnit)
			}
			if item.UnitPrice != tt.wantUnitPrice {
				t.Errorf("UnitPrice = %v; want %v", item.UnitPrice, tt.wantUnitPrice)
			}
			if item.TotalPrice != tt.wantTotal {
				t.Errorf("TotalPrice = %v; want %v", item.TotalPrice, tt.wantTotal)
			}
		})
	}
}

func TestParseSlip(t *testing.T) {
	rawSlip := `1. 2 kg cheeni 80 rs
2) 1 packet amul butter 55 rs
* 50 ka tel`

	slip, ok := parchi.ParseSlip(rawSlip)
	if !ok {
		t.Fatalf("expected ParseSlip to succeed with 100%% confidence, got ok=false")
	}
	if slip.TotalLinesParsed != 3 {
		t.Errorf("TotalLinesParsed = %d; want 3", slip.TotalLinesParsed)
	}
	if slip.MatchedCount != 3 {
		t.Errorf("MatchedCount = %d; want 3", slip.MatchedCount)
	}
	if slip.UnmatchedCount != 0 {
		t.Errorf("UnmatchedCount = %d; want 0", slip.UnmatchedCount)
	}
	expectedTotal := 80.0 + 55.0 + 50.0 // 185
	if slip.EstimatedTotal != expectedTotal {
		t.Errorf("EstimatedTotal = %v; want %v", slip.EstimatedTotal, expectedTotal)
	}

	// Test partial slip with conversational line
	partialSlip := `1. 2 kg atta
2. bhaiya kaunsa tel achha hai
3. 50 ka namak`
	pSlip, pOk := parchi.ParseSlip(partialSlip)
	if pOk {
		t.Errorf("expected partial slip to return false for 100%% confidence")
	}
	if pSlip.MatchedCount != 2 {
		t.Errorf("MatchedCount = %d; want 2", pSlip.MatchedCount)
	}
	if pSlip.UnmatchedCount != 1 {
		t.Errorf("UnmatchedCount = %d; want 1", pSlip.UnmatchedCount)
	}
}

func TestSplitCommands(t *testing.T) {
	input := "2 kg cheeni, 1 packet amul butter aur 50 ka tel + 1 dabba ghee\n2 pcs soap"
	cmds := parchi.SplitCommands(input)
	if len(cmds) != 5 {
		t.Fatalf("expected 5 commands, got %d: %#v", len(cmds), cmds)
	}
	expected := []string{
		"2 kg cheeni",
		"1 packet amul butter",
		"50 ka tel",
		"1 dabba ghee",
		"2 pcs soap",
	}
	for i, c := range cmds {
		if c != expected[i] {
			t.Errorf("cmds[%d] = %q; want %q", i, c, expected[i])
		}
	}
}

func TestNormalizeUnit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"kilo", "kg"},
		{"Kg", "kg"},
		{"pkt", "packet"},
		{"ltr", "litre"},
		{"boxes", "box"},
		{"bora", "sack"},
		{"darjan", "dozen"},
		{"", "pcs"},
		{"unknown_unit", "unknown_unit"},
	}

	for _, tt := range tests {
		got := parchi.NormalizeUnit(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeUnit(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestHasConversationalStopWords(t *testing.T) {
	if !parchi.HasConversationalStopWords("bhaiya rate kya hai") {
		t.Errorf("expected true for conversational query")
	}
	if parchi.HasConversationalStopWords("2 kg atta") {
		t.Errorf("expected false for pure retail query")
	}
}

func BenchmarkParseSingleLine(b *testing.B) {
	line := "1 packet amul butter 55 rs"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parchi.ParseSingleLine(line)
	}
}

func BenchmarkParseSlip(b *testing.B) {
	rawSlip := `1. 2 kg cheeni 80 rs
2) 1 packet amul butter 55 rs
* 50 ka tel`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parchi.ParseSlip(rawSlip)
	}
}

func BenchmarkSplitCommands(b *testing.B) {
	input := "2 kg cheeni, 1 packet amul butter aur 50 ka tel + 1 dabba ghee\n2 pcs soap"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parchi.SplitCommands(input)
	}
}
