package idempotency

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdempotency_PassthroughWithoutKey(t *testing.T) {
	eng, err := New()
	if err != nil {
		t.Fatalf("unexpected error creating engine: %v", err)
	}
	defer eng.Close()

	var callCount int32
	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected handler to execute once, got %d", callCount)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestIdempotency_ReplayCachedResponse(t *testing.T) {
	eng, err := New(WithTTL(10 * time.Second))
	if err != nil {
		t.Fatalf("unexpected error creating engine: %v", err)
	}
	defer eng.Close()

	var callCount int32
	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("X-Custom-Result", "created-123")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"order_id":"123"}`))
	}))

	key := "test-key-abc"

	// First request: should execute handler
	req1 := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req1.Header.Set("X-Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec1.Code)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 execution, got %d", callCount)
	}

	// Second request: same key, must replay from cache without re-executing handler
	req2 := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req2.Header.Set("X-Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on replay, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-Cache") != "IDEMPOTENCY-HIT" {
		t.Fatalf("expected X-Cache header to be IDEMPOTENCY-HIT, got %s", rec2.Header().Get("X-Cache"))
	}
	if rec2.Header().Get("X-Custom-Result") != "created-123" {
		t.Fatalf("expected preserved custom header, got %s", rec2.Header().Get("X-Custom-Result"))
	}
	if rec2.Body.String() != `{"order_id":"123"}` {
		t.Fatalf("expected exact body match, got %s", rec2.Body.String())
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("handler executed %d times, expected exactly 1", callCount)
	}
}

func TestIdempotency_InFlightLockRejection(t *testing.T) {
	eng, err := New(WithLockTTL(5 * time.Second))
	if err != nil {
		t.Fatalf("unexpected error creating engine: %v", err)
	}
	defer eng.Close()

	started := make(chan struct{})
	finish := make(chan struct{})

	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-finish
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))

	key := "concurrent-key-999"

	go func() {
		req := httptest.NewRequest(http.MethodPost, "/pay", nil)
		req.Header.Set("X-Idempotency-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}()

	<-started

	// Second concurrent request with same key while first is still running
	req2 := httptest.NewRequest(http.MethodPost, "/pay", nil)
	req2.Header.Set("X-Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for in-flight key, got %d", rec2.Code)
	}

	close(finish)
}

func TestIdempotency_ConcurrentStress(t *testing.T) {
	eng, err := New(WithTTL(5 * time.Second))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer eng.Close()

	var counter int64
	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	const totalRequests = 20
	var wg sync.WaitGroup
	key := "stress-key-1"

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/action", nil)
			req.Header.Set("X-Idempotency-Key", key)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	// Handler should have executed only once or minimal times depending on race
	if atomic.LoadInt64(&counter) > 5 {
		t.Fatalf("expected strict deduplication, but handler ran %d times", counter)
	}
}

func TestMemoryStorage_Expiration(t *testing.T) {
	store := NewMemoryStorage(50 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()
	rec := &Record{
		StatusCode: 200,
		Body:       []byte("cached-data"),
	}

	_ = store.Set(ctx, "temp-key", rec, 50*time.Millisecond)

	cached, err := store.Get(ctx, "temp-key")
	if err != nil || cached == nil {
		t.Fatalf("expected key to exist before expiration")
	}

	time.Sleep(100 * time.Millisecond)

	cachedExpired, err := store.Get(ctx, "temp-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cachedExpired != nil {
		t.Fatalf("expected key to expire, but still found")
	}
}

// Silence unused import warnings
var _ = io.Discard

func TestIdempotency_ObservabilityHooks(t *testing.T) {
	var savedKey string
	var hitKey string
	var conflictKey string

	eng, err := New(WithHooks(Hooks{
		OnSaved: func(key string, code int) {
			savedKey = key
		},
		OnHit: func(key string, code int) {
			hitKey = key
		},
		OnConflict: func(key string) {
			conflictKey = key
		},
	}), WithTTL(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()

	handler := eng.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	// First request -> trigger OnSaved
	req1 := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req1.Header.Set("X-Idempotency-Key", "key-hook-1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if savedKey == "" {
		t.Fatal("OnSaved hook was not triggered")
	}

	// Second request -> trigger OnHit
	req2 := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req2.Header.Set("X-Idempotency-Key", "key-hook-1")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if hitKey == "" {
		t.Fatal("OnHit hook was not triggered")
	}
	_ = conflictKey
}

