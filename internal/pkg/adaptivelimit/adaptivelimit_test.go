package adaptivelimit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"shopMe/internal/pkg/adaptivelimit"
)

func TestLimiterAcquireAndRelease(t *testing.T) {
	limiter := adaptivelimit.New(adaptivelimit.WithInitialLimit(5))
	defer limiter.Close()

	ctx := context.Background()
	release, err := limiter.Acquire(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := limiter.Metrics()
	if m.InFlight != 1 {
		t.Fatalf("expected inFlight 1, got %d", m.InFlight)
	}

	release()

	m = limiter.Metrics()
	if m.InFlight != 0 {
		t.Fatalf("expected inFlight 0 after release, got %d", m.InFlight)
	}
}

func TestLimitExceededRejection(t *testing.T) {
	// Initial limit 2
	limiter := adaptivelimit.New(adaptivelimit.WithInitialLimit(2))
	defer limiter.Close()

	ctx := context.Background()

	rel1, err1 := limiter.Acquire(ctx)
	if err1 != nil {
		t.Fatalf("err1 unexpected: %v", err1)
	}
	defer rel1()

	rel2, err2 := limiter.Acquire(ctx)
	if err2 != nil {
		t.Fatalf("err2 unexpected: %v", err2)
	}
	defer rel2()

	// 3rd acquire must fail with ErrLimitExceeded
	_, err3 := limiter.Acquire(ctx)
	if !errors.Is(err3, adaptivelimit.ErrLimitExceeded) {
		t.Fatalf("expected ErrLimitExceeded, got %v", err3)
	}
}

func TestAdditiveIncreaseUnderHealthyLoad(t *testing.T) {
	limiter := adaptivelimit.New(
		adaptivelimit.WithInitialLimit(5),
		adaptivelimit.WithLimits(2, 20),
	)
	defer limiter.Close()

	initial := limiter.Limit()

	// Run fast operations to establish low RTT and low queue
	for i := 0; i < 5; i++ {
		release, err := limiter.Acquire(context.Background())
		if err != nil {
			t.Fatalf("unexpected acquire failure: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
		release()
	}

	current := limiter.Limit()
	if current <= initial {
		t.Fatalf("expected limit to increase under healthy load, initial=%f current=%f", initial, current)
	}
}

func TestMultiplicativeDecreaseOnLatencySpike(t *testing.T) {
	limiter := adaptivelimit.New(
		adaptivelimit.WithInitialLimit(10),
		adaptivelimit.WithThresholds(1.0, 2.0),
		adaptivelimit.WithBackoffRatio(0.5),
	)
	defer limiter.Close()

	// 1. Establish baseline minRTT of 1ms
	rel, _ := limiter.Acquire(context.Background())
	time.Sleep(1 * time.Millisecond)
	rel()

	expandedLimit := limiter.Limit()

	// 2. Now simulate severe downstream database latency spike (50ms)
	relSpike, _ := limiter.Acquire(context.Background())
	time.Sleep(50 * time.Millisecond)
	relSpike()

	decreasedLimit := limiter.Limit()
	if decreasedLimit >= expandedLimit {
		t.Fatalf("expected limit to drop on latency spike, expanded=%f decreased=%f", expandedLimit, decreasedLimit)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	limiter := adaptivelimit.New(adaptivelimit.WithInitialLimit(1))
	defer limiter.Close()

	middleware := adaptivelimit.Middleware(limiter)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	// First request succeeds
	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		close(done)
	}()

	time.Sleep(5 * time.Millisecond)

	// Second concurrent request should be rejected with 503
	req2 := httptest.NewRequest("GET", "/test", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 status code when limit exceeded, got %d", rr2.Code)
	}

	<-done
}

