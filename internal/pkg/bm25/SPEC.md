# SPEC: go-bm25

## 1. Scope & Purpose
`go-bm25` provides probabilistic full-text search and relevance ranking for Go applications. It handles tokenization, inverted indexing, term frequency saturation, and document length normalization.

It does NOT handle disk-backed distributed shards (use Elasticsearch for multi-terabyte cluster storage).

## 2. Public API Contract
```go
type Document struct { ID, Text string, Data any }
type SearchResult struct { DocID string, Score float64, Data any }
type Index struct { ... }

func New(opts ...Option) *Index
func (idx *Index) Add(doc Document) error
func (idx *Index) AddBatch(docs []Document) error
func (idx *Index) Delete(id string) error
func (idx *Index) Search(query string, topK int) []SearchResult
func (idx *Index) Count() int
```

## 3. Concurrency & Memory Guarantees
- Thread-safe via `sync.RWMutex` (parallel read searches with concurrent write locking).
- Inverted index stores integer postings mappings.
- Clean memory reclamation on document deletion.

## 4. How to Extend
- Add Porter Stemming / N-gram tokenizer filters.
- Add field-specific boosting (e.g. title weight $\times 2.0$, description weight $\times 1.0$).
