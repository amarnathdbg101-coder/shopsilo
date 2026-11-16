# go-bm25 🔍

> **High-Performance In-Memory Full-Text Search Engine powered by Okapi BM25 Ranking for Go (Go 1.23+).**  
> Provides Elasticsearch / Lucene-grade probabilistic relevance ranking, document length normalization, and inverted indexing in pure Go with zero external dependencies.

---

## 🚀 Key Features

1. **Mathematically Proven Okapi BM25**:
   - Implements the canonical Robertson-Spärck Jones IDF and non-linear term frequency saturation formula:
     $$\text{Score}(D, Q) = \sum_{t \in Q} \text{IDF}(t) \cdot \frac{f(t, D) \cdot (k_1 + 1)}{f(t, D) + k_1 \cdot \left(1 - b + b \cdot \frac{|D|}{\text{avgdl}}\right)}$$
2. **Document Length Normalization ($b$)**:
   - Automatically penalizes diluted, excessively verbose documents while boosting concise, highly targeted matches.
3. **In-Memory Inverted Index**:
   - Thread-safe Concurrent Inverted Postings with sub-millisecond retrieval across tens of thousands of documents.
4. **Zero Dependencies**:
   - 100% pure standard library (`math`, `strings`, `unicode`, `sync`, `sort`).

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-bm25"
)

func main() {
	idx := bm25.New(bm25.WithK1(1.5), bm25.WithB(0.75))

	// Index products or documents
	_ = idx.Add(bm25.Document{ID: "p1", Text: "Basmati Rice 5kg Premium Long Grain"})
	_ = idx.Add(bm25.Document{ID: "p2", Text: "Fortune Sunlite Refined Sunflower Oil 1L"})
	_ = idx.Add(bm25.Document{ID: "p3", Text: "Daawat Rozana Super Basmati Rice 1kg"})

	// Search by relevance
	results := idx.Search("Basmati Rice", 5)
	for _, r := range results {
		fmt.Printf("Rank #%s (Score: %.4f)\n", r.DocID, r.Score)
	}
}
```
