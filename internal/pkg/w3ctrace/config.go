package w3ctrace

import "sync"

// Exporter processes completed spans.
type Exporter interface {
	Export(span *Span) error
}

// MemoryExporter buffers exported spans in memory.
type MemoryExporter struct {
	mu    sync.Mutex
	spans []*Span
}

// NewMemoryExporter creates an in-memory exporter.
func NewMemoryExporter() *MemoryExporter {
	return &MemoryExporter{
		spans: make([]*Span, 0),
	}
}

// Export stores a completed span.
func (e *MemoryExporter) Export(span *Span) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, span)
	return nil
}

// Spans returns a copy of all buffered spans.
func (e *MemoryExporter) Spans() []*Span {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]*Span, len(e.spans))
	copy(cp, e.spans)
	return cp
}

// Clear flushes all buffered spans.
func (e *MemoryExporter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = nil
}
