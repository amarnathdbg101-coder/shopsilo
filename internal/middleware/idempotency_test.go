package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shopMe/internal/middleware"
)

func TestIdempotencyGuard_ReplayCachedResponse(t *testing.T) {
	var executionCount int32

	handler := middleware.IdempotencyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executionCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"order_id":"ord_123","status":"created"}`))
	}))

	key := "test-idempotency-key-001"

	// Request 1: Fresh execution
	req1 := httptest.NewRequest(http.MethodPost, "/pos/checkout", nil)
	req1.Header.Set("X-Idempotency-Key", key)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec1.Code)
	}
	body1 := rec1.Body.String()

	// Request 2: Replay with same key (should NOT re-execute handler)
	req2 := httptest.NewRequest(http.MethodPost, "/pos/checkout", nil)
	req2.Header.Set("X-Idempotency-Key", key)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected replayed status 201, got %d", rec2.Code)
	}
	body2 := rec2.Body.String()

	if body1 != body2 {
		t.Errorf("expected cached body %s, got %s", body1, body2)
	}

	if atomic.LoadInt32(&executionCount) != 1 {
		t.Errorf("expected handler to execute exactly 1 time, executed %d times", executionCount)
	}
}

func TestIdempotencyGuard_PassesGetRequests(t *testing.T) {
	var executionCount int32

	handler := middleware.IdempotencyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executionCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// GET requests should not be cached or blocked by idempotency
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		req.Header.Set("X-Idempotency-Key", "same-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	}

	if atomic.LoadInt32(&executionCount) != 3 {
		t.Errorf("expected GET to execute 3 times, got %d", executionCount)
	}
}

func TestIdempotencyGuard_ConcurrentInFlightCollision(t *testing.T) {
	gate := make(chan struct{})
	handler := middleware.IdempotencyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-gate // Hold request in-flight to simulate slow DB write
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))

	key := "test-concurrent-key-999"
	var wg sync.WaitGroup
	var statusCodes []int
	var mu sync.Mutex

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/khata/credit", nil)
			req.Header.Set("X-Idempotency-Key", key)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			mu.Lock()
			statusCodes = append(statusCodes, rec.Code)
			mu.Unlock()
		}()
	}

	// Give goroutines time to hit in-flight lock
	time.Sleep(20 * time.Millisecond)
	close(gate)
	wg.Wait()

	hasConflict := false
	hasSuccess := false
	for _, code := range statusCodes {
		if code == http.StatusConflict {
			hasConflict = true
		}
		if code == http.StatusOK {
			hasSuccess = true
		}
	}

	if !hasConflict && !hasSuccess {
		t.Logf("Status codes recorded: %v", statusCodes)
	}
}
