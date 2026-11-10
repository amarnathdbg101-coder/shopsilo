package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shopMe/internal/middleware"
)

func TestTraceMiddleware_InjectsW3CTraceparentHeader(t *testing.T) {
	handler := middleware.TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	traceparent := rec.Header().Get("traceparent")
	if traceparent == "" {
		t.Fatal("expected traceparent header to be present in response")
	}

	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts in W3C traceparent (version-traceID-parentID-flags), got %q", traceparent)
	}

	if parts[0] != "00" {
		t.Errorf("expected W3C version '00', got %q", parts[0])
	}
	if len(parts[1]) != 32 {
		t.Errorf("expected 32-char hex traceID, got %q", parts[1])
	}
	if len(parts[2]) != 16 {
		t.Errorf("expected 16-char hex parentID, got %q", parts[2])
	}
}
