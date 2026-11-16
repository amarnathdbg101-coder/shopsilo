package gohealthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}

func TestHealth_AllHealthy(t *testing.T) {
	h := New(Config{Timeout: time.Second, CacheTTL: 0})

	h.RegisterReadiness(NewPingChecker("postgres", &mockPinger{err: nil}))
	h.RegisterReadiness(NewCheckFunc("redis", func(ctx context.Context) error {
		return nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	h.ReadyHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var report Report
	if err := json.NewDecoder(rr.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode json report: %v", err)
	}

	if report.Status != StatusUp {
		t.Fatalf("expected overall status UP, got %s", report.Status)
	}
	if report.Details["postgres"].Status != StatusUp {
		t.Fatalf("expected postgres UP, got %s", report.Details["postgres"].Status)
	}
	if report.Details["redis"].Status != StatusUp {
		t.Fatalf("expected redis UP, got %s", report.Details["redis"].Status)
	}
}

func TestHealth_DependencyDown_Returns503(t *testing.T) {
	h := New(Config{Timeout: time.Second, CacheTTL: 0})

	// Postgres fails
	h.RegisterReadiness(NewPingChecker("postgres", &mockPinger{err: errors.New("connection refused")}))
	h.RegisterReadiness(NewCheckFunc("redis", func(ctx context.Context) error {
		return nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	h.ReadyHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var report Report
	_ = json.NewDecoder(rr.Body).Decode(&report)

	if report.Status != StatusDown {
		t.Fatalf("expected overall status DOWN, got %s", report.Status)
	}
	if report.Details["postgres"].Status != StatusDown {
		t.Fatal("expected postgres to be marked DOWN")
	}
	if report.Details["postgres"].Error != "connection refused" {
		t.Fatalf("expected error message 'connection refused', got %s", report.Details["postgres"].Error)
	}
}

func TestHealth_CacheTTL(t *testing.T) {
	var checkCount int32
	h := New(Config{
		Timeout:  time.Second,
		CacheTTL: 100 * time.Millisecond,
	})

	h.RegisterReadiness(NewCheckFunc("counter_check", func(ctx context.Context) error {
		atomic.AddInt32(&checkCount, 1)
		return nil
	}))

	ctx := context.Background()
	// Call 1
	_ = h.CheckReadiness(ctx)
	// Call 2 immediately (should hit cache)
	_ = h.CheckReadiness(ctx)
	// Call 3 immediately (should hit cache)
	_ = h.CheckReadiness(ctx)

	if atomic.LoadInt32(&checkCount) != 1 {
		t.Fatalf("expected 1 execution due to caching, got %d", atomic.LoadInt32(&checkCount))
	}

	// Wait for cache expiration
	time.Sleep(120 * time.Millisecond)
	_ = h.CheckReadiness(ctx)

	if atomic.LoadInt32(&checkCount) != 2 {
		t.Fatalf("expected 2 executions after TTL expired, got %d", atomic.LoadInt32(&checkCount))
	}
}

func TestHealth_TimeoutEnforcement(t *testing.T) {
	h := New(Config{
		Timeout:  50 * time.Millisecond,
		CacheTTL: 0,
	})

	// Slow checker that hangs
	h.RegisterReadiness(NewCheckFunc("slow_db", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	h.ReadyHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on timeout, got %d", rr.Code)
	}

	var report Report
	_ = json.NewDecoder(rr.Body).Decode(&report)
	if report.Status != StatusDown {
		t.Fatalf("expected DOWN status on timeout, got %s", report.Status)
	}
}

func TestHealth_MemoryChecker(t *testing.T) {
	// Require max 1 byte (impossible, will definitely fail)
	memChecker := NewMemoryChecker("memory_leak_test", 1)
	err := memChecker.Check(context.Background())
	if err == nil {
		t.Fatal("expected memory threshold to fail")
	}

	// 500 MB limit (should pass easily in unit test)
	memPass := NewMemoryChecker("memory_pass_test", 500*1024*1024)
	if err := memPass.Check(context.Background()); err != nil {
		t.Fatalf("expected memory check to pass, got %v", err)
	}
}
