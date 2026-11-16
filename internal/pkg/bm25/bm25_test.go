package bm25_test

import (
	"testing"

	"shopMe/internal/pkg/bm25"
)

func TestBM25BasicRelevance(t *testing.T) {
	idx := bm25.New()

	docs := []bm25.Document{
		{ID: "doc1", Text: "Go is an open source programming language for building fast systems"},
		{ID: "doc2", Text: "Python is a dynamic programming language popular for machine learning"},
		{ID: "doc3", Text: "Go concurrency with goroutines and channels makes Go extremely fast"},
	}

	if err := idx.AddBatch(docs); err != nil {
		t.Fatalf("failed to add docs: %v", err)
	}

	if idx.Count() != 3 {
		t.Fatalf("expected count 3, got %d", idx.Count())
	}

	// Search for "Go concurrency fast"
	results := idx.Search("Go concurrency fast", 2)
	if len(results) == 0 {
		t.Fatalf("expected search results, got 0")
	}

	// doc3 mentions "Go" twice, "concurrency", and "fast", so it should score highest!
	if results[0].DocID != "doc3" {
		t.Errorf("expected doc3 to rank #1, got %s with score %f", results[0].DocID, results[0].Score)
	}
}

func TestBM25DocumentLengthPenalty(t *testing.T) {
	idx := bm25.New(bm25.WithB(0.75))

	// Short document with term vs long document where term is diluted
	docShort := bm25.Document{ID: "short", Text: "PostgreSQL database"}
	docLong := bm25.Document{
		ID:   "long",
		Text: "The user searched for something else in the web application and then briefly queried the PostgreSQL database before logging out and shutting down the terminal window",
	}

	_ = idx.Add(docShort)
	_ = idx.Add(docLong)

	results := idx.Search("PostgreSQL", 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Short doc should rank higher due to BM25 length normalization
	if results[0].DocID != "short" {
		t.Errorf("expected 'short' doc to score higher than diluted 'long' doc, got %s", results[0].DocID)
	}
}

func TestDeleteAndReindex(t *testing.T) {
	idx := bm25.New()

	_ = idx.Add(bm25.Document{ID: "101", Text: "Amardeep ShopSilo backend"})
	results := idx.Search("ShopSilo", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if err := idx.Delete("101"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	resultsAfter := idx.Search("ShopSilo", 10)
	if len(resultsAfter) != 0 {
		t.Fatalf("deleted document should not be returned in search")
	}
}

