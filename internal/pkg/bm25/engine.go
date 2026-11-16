package bm25

import (
	"math"
	"sort"
	"sync"
)

// Document represents an indexed entity with optional user metadata.
type Document struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Data any    `json:"data,omitempty"`
}

// SearchResult represents a ranked matching document.
type SearchResult struct {
	DocID string  `json:"doc_id"`
	Score float64 `json:"score"`
	Data  any     `json:"data,omitempty"`
}

// Index represents an in-memory full-text search index powered by Okapi BM25 ranking.
type Index struct {
	cfg Config
	mu  sync.RWMutex

	docs        map[string]Document
	docLengths  map[string]int
	inverted    map[string]map[string]int // term -> docID -> termFrequency
	totalTokens int
}

// New creates a new BM25 index instance with optional tuning.
func New(opts ...Option) *Index {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Index{
		cfg:        cfg,
		docs:       make(map[string]Document),
		docLengths: make(map[string]int),
		inverted:   make(map[string]map[string]int),
	}
}

// Add indexes a single document.
func (idx *Index) Add(doc Document) error {
	if doc.ID == "" || doc.Text == "" {
		return ErrEmptyDocument
	}

	tokens := Tokenize(doc.Text, idx.cfg.MinTokenLength)
	if len(tokens) == 0 {
		return ErrEmptyDocument
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// If document already exists, remove it first
	if _, exists := idx.docs[doc.ID]; exists {
		idx.deleteLocked(doc.ID)
	}

	idx.docs[doc.ID] = doc
	docLen := len(tokens)
	idx.docLengths[doc.ID] = docLen
	idx.totalTokens += docLen

	for _, token := range tokens {
		if _, ok := idx.inverted[token]; !ok {
			idx.inverted[token] = make(map[string]int)
		}
		idx.inverted[token][doc.ID]++
	}

	return nil
}

// AddBatch indexes multiple documents.
func (idx *Index) AddBatch(docs []Document) error {
	for _, d := range docs {
		if err := idx.Add(d); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a document from the index.
func (idx *Index) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.docs[id]; !exists {
		return ErrDocumentNotFound
	}
	idx.deleteLocked(id)
	return nil
}

func (idx *Index) deleteLocked(id string) {
	docLen := idx.docLengths[id]
	idx.totalTokens -= docLen
	delete(idx.docLengths, id)
	delete(idx.docs, id)

	for token, docMap := range idx.inverted {
		delete(docMap, id)
		if len(docMap) == 0 {
			delete(idx.inverted, token)
		}
	}
}

// Count returns the number of documents in the index.
func (idx *Index) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.docs)
}

// Search ranks documents matching the query using the Okapi BM25 formula.
func (idx *Index) Search(query string, topK int) []SearchResult {
	if topK <= 0 {
		topK = 10
	}

	queryTokens := Tokenize(query, idx.cfg.MinTokenLength)
	if len(queryTokens) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	totalDocs := len(idx.docs)
	if totalDocs == 0 {
		return nil
	}

	avgdl := float64(idx.totalTokens) / float64(totalDocs)
	scores := make(map[string]float64)

	for _, qTerm := range queryTokens {
		posting, exists := idx.inverted[qTerm]
		if !exists {
			continue
		}

		nDocWithTerm := len(posting)
		// Robertson-Spärck Jones IDF: ln(1 + (N - n + 0.5) / (n + 0.5))
		idf := math.Log(1.0 + (float64(totalDocs)-float64(nDocWithTerm)+0.5)/(float64(nDocWithTerm)+0.5))
		if idf < 0 {
			idf = 0.0001
		}

		for docID, tf := range posting {
			docLen := float64(idx.docLengths[docID])
			// BM25 term score component
			numerator := float64(tf) * (idx.cfg.K1 + 1.0)
			denominator := float64(tf) + idx.cfg.K1*(1.0-idx.cfg.B+idx.cfg.B*(docLen/avgdl))
			scores[docID] += idf * (numerator / denominator)
		}
	}

	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		doc := idx.docs[docID]
		results = append(results, SearchResult{
			DocID: docID,
			Score: score,
			Data:  doc.Data,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}
