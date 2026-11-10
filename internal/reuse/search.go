// Package reuse provides reusable foundational wrappers and shared infrastructure utilities.
package reuse

import (
	"github.com/amarnathdbg101-coder/go/go-bm25"
)

// ProductSearchItem represents a product indexed for in-memory BM25 ranking.
type ProductSearchItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Brand       string `json:"brand"`
}

// ProductSearchEngine wraps Okapi BM25 ranking for sub-millisecond product retrieval.
type ProductSearchEngine struct {
	idx *bm25.Index
}

// NewProductSearchEngine creates a new BM25 product search engine.
func NewProductSearchEngine() *ProductSearchEngine {
	return &ProductSearchEngine{
		idx: bm25.New(
			bm25.WithK1(1.5),
			bm25.WithB(0.75),
		),
	}
}

// IndexProduct adds or updates a product in the BM25 search index.
func (e *ProductSearchEngine) IndexProduct(item ProductSearchItem) error {
	fullText := item.Title + " " + item.Category + " " + item.Brand + " " + item.Description
	return e.idx.Add(bm25.Document{
		ID:   item.ID,
		Text: fullText,
		Data: item,
	})
}

// Search queries the BM25 index and returns top matching product IDs ranked by relevance.
func (e *ProductSearchEngine) Search(query string, topK int) []bm25.SearchResult {
	if e == nil || e.idx == nil {
		return nil
	}
	return e.idx.Search(query, topK)
}

// GlobalProductSearchEngine is the singleton in-memory search index for high-speed product queries.
var GlobalProductSearchEngine = NewProductSearchEngine()
