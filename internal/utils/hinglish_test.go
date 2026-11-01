package utils

import (
	"testing"
)

func TestHinglishExpansion(t *testing.T) {
	tests := []struct {
		query    string
		minTerms int
		contains string
	}{
		{"doodh", 4, "milk"},
		{"dudh", 3, "milk"},
		{"milk", 4, "doodh"},
		{"amul doodh", 2, "milk"},
		{"tel", 6, "oil"},
		{"cheeni", 5, "sugar"},
		{"chawal", 4, "rice"},
		{"aata", 5, "flour"},
		{"sabun", 5, "soap"},
		{"dawa", 6, "medicine"},
		{"aloo", 3, "potato"},
		{"pyaz", 3, "onion"},
		{"chai", 4, "tea"},
		{"biskut", 3, "biscuit"},
		{"cold drink", 7, "beverage"},
	}

	for _, tt := range tests {
		groups := ExpandHinglishSearchGroups(tt.query)
		if len(groups) == 0 {
			t.Errorf("expected groups for %q, got 0", tt.query)
			continue
		}

		found := false
		for _, g := range groups {
			for _, term := range g.Terms {
				if term == tt.contains {
					found = true
					break
				}
			}
		}

		if !found {
			t.Errorf("query %q: expected expansion to contain %q, but groups were %+v", tt.query, tt.contains, groups)
		}
	}
}

func TestHinglishStopwordStripping(t *testing.T) {
	t.Run("Strips 'ka' stopword from conversational search", func(t *testing.T) {
		groups := ExpandHinglishSearchGroups("amul ka doodh")
		if len(groups) != 2 {
			t.Fatalf("expected 2 search groups (amul, doodh), got %d: %+v", len(groups), groups)
		}
		// Group 1 should be amul
		if len(groups[0].Terms) != 1 || groups[0].Terms[0] != "amul" {
			t.Errorf("group 1 expected ['amul'], got %v", groups[0].Terms)
		}
		// Group 2 should expand doodh -> milk
		hasMilk := false
		for _, term := range groups[1].Terms {
			if term == "milk" {
				hasMilk = true
			}
		}
		if !hasMilk {
			t.Errorf("group 2 expected to contain 'milk', got %v", groups[1].Terms)
		}
	})

	t.Run("Strips 'chahiye' and 'sabse sasta' fillers", func(t *testing.T) {
		groups := ExpandHinglishSearchGroups("sabse sasta tel chahiye")
		if len(groups) != 1 {
			t.Fatalf("expected 1 group (tel), got %d: %+v", len(groups), groups)
		}
		hasOil := false
		for _, term := range groups[0].Terms {
			if term == "oil" {
				hasOil = true
			}
		}
		if !hasOil {
			t.Errorf("expected group to contain 'oil', got %v", groups[0].Terms)
		}
	})
}

func TestHinglishPhoneticNormalization(t *testing.T) {
	t.Run("Phonetic 'zeera' matches 'jeera' / 'cumin'", func(t *testing.T) {
		groups := ExpandHinglishSearchGroups("zeera")
		if len(groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(groups))
		}
		hasCumin := false
		for _, term := range groups[0].Terms {
			if term == "cumin" {
				hasCumin = true
			}
		}
		if !hasCumin {
			t.Errorf("expected zeera to expand to cumin, got %v", groups[0].Terms)
		}
	})
}
