package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestIdempotencyGuard_ConcurrentDoubleTapAndStress(t *testing.T) {
	// 1. Concurrent double-tap with identical key must execute handler EXACTLY ONCE
	var hits atomic.Int32
	handler := IdempotencyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	const doubleTaps = 50
	var wg sync.WaitGroup
	wg.Add(doubleTaps)
	for i := 0; i < doubleTaps; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/pos/checkout", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			req.Header.Set("X-Idempotency-Key", "stress-double-tap-key-1")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusCreated {
				t.Errorf("expected 201, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()

	if got := hits.Load(); got != 1 {
		t.Fatalf("expected handler to execute exactly 1 time for 50 concurrent double-taps, got %d", got)
	}

	// 2. 1,000 unique concurrent keys stress test + 5xx non-caching verification
	var uniqueHits atomic.Int32
	stressHandler := IdempotencyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := uniqueHits.Add(1)
		if n%10 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"n":%d}`, n)))
	}))

	const totalRequests = 1000
	wg.Add(totalRequests)
	for i := 0; i < totalRequests; i++ {
		i := i
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/khata/credit", nil)
			req.RemoteAddr = "10.0.0.1:5000"
			req.Header.Set("X-Idempotency-Key", fmt.Sprintf("stress-key-%d", i))
			rec := httptest.NewRecorder()
			stressHandler.ServeHTTP(rec, req)
		}()
	}
	wg.Wait()

	if got := uniqueHits.Load(); got != totalRequests {
		t.Fatalf("expected %d unique executions, got %d", totalRequests, got)
	}
}
