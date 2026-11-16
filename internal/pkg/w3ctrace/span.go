package w3ctrace

import (
	"context"
	"sync"
	"time"
)

type traceCtxKey struct{}

// Span represents an active or completed unit of work within a distributed trace.
type Span struct {
	Context      SpanContext       `json:"context"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	ServiceName  string            `json:"service_name,omitempty"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Attributes   map[string]string `json:"attributes"`
	Status       string            `json:"status"` // "OK" or "ERROR"
	Error        string            `json:"error,omitempty"`

	mu     sync.Mutex
	ended  bool
	tracer *Tracer
}

// Tracer orchestrates span creation and lifecycle management.
type Tracer struct {
	serviceName string
	exporter    Exporter
}

var globalTracer = &Tracer{
	serviceName: "service",
	exporter:    NewMemoryExporter(),
}

// SetGlobalTracer sets the default tracer instance.
func SetGlobalTracer(t *Tracer) {
	if t != nil {
		globalTracer = t
	}
}

// GetGlobalTracer returns the active global tracer.
func GetGlobalTracer() *Tracer {
	return globalTracer
}

// NewTracer creates a new Tracer instance with service name and exporter.
func NewTracer(serviceName string, exporter Exporter) *Tracer {
	if exporter == nil {
		exporter = NewMemoryExporter()
	}
	return &Tracer{
		serviceName: serviceName,
		exporter:    exporter,
	}
}

// StartSpan creates a new child span from context or starts a new root trace.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	parent := SpanFromContext(ctx)

	var traceID, parentSpanID string
	var traceState string

	if parent != nil && parent.Context.IsValid() {
		traceID = parent.Context.TraceID
		parentSpanID = parent.Context.SpanID
		traceState = parent.Context.TraceState
	} else {
		traceID = NewTraceID()
	}

	spanID := NewSpanID()

	sc := SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: FlagSampled,
		TraceState: traceState,
	}

	span := &Span{
		Context:      sc,
		ParentSpanID: parentSpanID,
		Name:         name,
		ServiceName:  t.serviceName,
		StartTime:    time.Now(),
		Attributes:   make(map[string]string),
		Status:       "OK",
		tracer:       t,
	}

	return context.WithValue(ctx, traceCtxKey{}, span), span
}

// StartSpan is a convenience function using the global tracer.
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	return globalTracer.StartSpan(ctx, name)
}

// SpanFromContext extracts the active span from context, returning nil if absent.
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	if span, ok := ctx.Value(traceCtxKey{}).(*Span); ok {
		return span
	}
	return nil
}

// SetAttribute sets a string key-value attribute on the span.
func (s *Span) SetAttribute(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.Attributes[key] = value
}

// RecordError flags the span as an error and logs the message.
func (s *Span) RecordError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.Status = "ERROR"
	s.Error = err.Error()
	s.Attributes["error.message"] = err.Error()
}

// End finishes the span, calculates duration, and sends it to the exporter.
func (s *Span) End() {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	tracer := s.tracer
	s.mu.Unlock()

	if tracer != nil && tracer.exporter != nil {
		_ = tracer.exporter.Export(s)
	}
}
