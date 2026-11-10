package reuse_test

import (
	"testing"

	"shopMe/internal/reuse"
)

func TestProductSearchEngine_BM25Ranking(t *testing.T) {
	engine := reuse.NewProductSearchEngine()

	_ = engine.IndexProduct(reuse.ProductSearchItem{
		ID:       "p1",
		Title:    "Fortune Sunlite Refined Sunflower Oil 1L",
		Category: "Cooking Oil",
		Brand:    "Fortune",
	})
	_ = engine.IndexProduct(reuse.ProductSearchItem{
		ID:       "p2",
		Title:    "Daawat Rozana Super Basmati Rice 5kg",
		Category: "Rice & Grains",
		Brand:    "Daawat",
	})
	_ = engine.IndexProduct(reuse.ProductSearchItem{
		ID:       "p3",
		Title:    "India Gate Classic Basmati Rice 1kg",
		Category: "Rice & Grains",
		Brand:    "India Gate",
	})

	// Search for "Basmati Rice"
	results := engine.Search("Basmati Rice", 5)

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results for 'Basmati Rice', got %d", len(results))
	}

	// P2 and P3 must rank above P1 (Cooking Oil)
	for _, r := range results {
		if r.DocID == "p1" {
			t.Fatalf("oil product p1 should not be matched for 'Basmati Rice'")
		}
	}

	if results[0].Score <= 0 {
		t.Errorf("expected positive BM25 score, got %f", results[0].Score)
	}
}
