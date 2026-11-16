package w3ctrace_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"shopMe/internal/pkg/w3ctrace"
)

func TestParseValidTraceparent(t *testing.T) {
	raw := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	sc, err := w3ctrace.ParseTraceparent(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if sc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected TraceID: %s", sc.TraceID)
	}
	if sc.SpanID != "00f067aa0ba902b7" {
		t.Errorf("unexpected SpanID: %s", sc.SpanID)
	}
	if sc.TraceFlags != "01" {
		t.Errorf("unexpected TraceFlags: %s", sc.TraceFlags)
	}

	// Roundtrip serialization
	if sc.String() != raw {
		t.Errorf("expected %s, got %s", raw, sc.String())
	}
}

func TestParseInvalidTraceparent(t *testing.T) {
	// All zeros trace ID is forbidden per RFC
	invalidZeros := "00-00000000000000000000000000000000-00f067aa0ba902b7-01"
	if _, err := w3ctrace.ParseTraceparent(invalidZeros); err == nil {
		t.Errorf("expected error for all-zeros trace ID")
	}

	// Short span ID
	invalidSpan := "00-4bf92f3577b34da6a3ce929d0e0e4736-1234-01"
	if _, err := w3ctrace.ParseTraceparent(invalidSpan); err == nil {
		t.Errorf("expected error for short span ID")
	}
}

func TestSpanHierarchy(t *testing.T) {
	exporter := w3ctrace.NewMemoryExporter()
	tracer := w3ctrace.NewTracer("test-service", exporter)

	ctx := context.Background()

	// Root span
	ctx, rootSpan := tracer.StartSpan(ctx, "root-operation")
	time.Sleep(5 * time.Millisecond)

	// Child span
	_, childSpan := tracer.StartSpan(ctx, "child-db-query")
	childSpan.SetAttribute("db.statement", "SELECT * FROM users")
	time.Sleep(5 * time.Millisecond)
	childSpan.End()

	rootSpan.End()

	spans := exporter.Spans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 exported spans, got %d", len(spans))
	}

	childExported := spans[0]
	rootExported := spans[1]

	// Child must inherit root's TraceID
	if childExported.Context.TraceID != rootExported.Context.TraceID {
		t.Errorf("child TraceID %s != root TraceID %s", childExported.Context.TraceID, rootExported.Context.TraceID)
	}

	// Child's ParentSpanID must match root's SpanID
	if childExported.ParentSpanID != rootExported.Context.SpanID {
		t.Errorf("child ParentSpanID %s != root SpanID %s", childExported.ParentSpanID, rootExported.Context.SpanID)
	}

	if childExported.Attributes["db.statement"] != "SELECT * FROM users" {
		t.Errorf("missing attribute on child span")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	exporter := w3ctrace.NewMemoryExporter()
	tracer := w3ctrace.NewTracer("api-gateway", exporter)

	middleware := w3ctrace.Middleware(tracer)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := w3ctrace.SpanFromContext(r.Context())
		if span == nil {
			t.Errorf("expected active span in request context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Upstream request with incoming traceparent
	incomingTrace := "00-11112222333344445555666677778888-aaaaaaaaaaaaaaaa-01"
	req := httptest.NewRequest("GET", "/api/v1/orders", nil)
	req.Header.Set(w3ctrace.HeaderTraceparent, incomingTrace)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Response header must have traceparent
	respTrace := rr.Header().Get(w3ctrace.HeaderTraceparent)
	if respTrace == "" {
		t.Fatalf("expected traceparent header in response")
	}

	parsed, err := w3ctrace.ParseTraceparent(respTrace)
	if err != nil {
		t.Fatalf("failed to parse response traceparent: %v", err)
	}

	// TraceID must match upstream
	if parsed.TraceID != "11112222333344445555666677778888" {
		t.Errorf("expected trace ID preserved, got %s", parsed.TraceID)
	}

	spans := exporter.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	if spans[0].Attributes["http.method"] != "GET" {
		t.Errorf("expected http.method GET, got %s", spans[0].Attributes["http.method"])
	}
}

func TestRecordError(t *testing.T) {
	exporter := w3ctrace.NewMemoryExporter()
	tracer := w3ctrace.NewTracer("worker", exporter)

	_, span := tracer.StartSpan(context.Background(), "task")
	span.RecordError(errors.New("disk full"))
	span.End()

	spans := exporter.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status != "ERROR" || spans[0].Error != "disk full" {
		t.Errorf("expected ERROR status with 'disk full', got %s (%s)", spans[0].Status, spans[0].Error)
	}
}

