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
