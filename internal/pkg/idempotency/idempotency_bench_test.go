package idempotency

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkIdempotency_CacheHitReplay(b *testing.B) {
	eng, err := New(WithTTL(10 * time.Minute))
	if err != nil {
		b.Fatal(err)
	}
	defer eng.Close()

	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order_id":"123","status":"confirmed"}`))
	}))

	// Seed cache with first request
	reqSeed := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	reqSeed.Header.Set("X-Idempotency-Key", "bench-hit-key-1")
	recSeed := httptest.NewRecorder()
	handler.ServeHTTP(recSeed, reqSeed)

	// Benchmark hits
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.Header.Set("X-Idempotency-Key", "bench-hit-key-1")
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkIdempotency_PassthroughGET(b *testing.B) {
	eng, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer eng.Close()

	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rec, req)
	}
}
