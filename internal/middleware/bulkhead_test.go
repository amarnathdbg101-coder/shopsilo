package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"shopMe/internal/middleware"

	"github.com/amarnathdbg101-coder/go/go-bulkhead"
)

func TestBulkheadGuard_RejectsExcessConcurrency(t *testing.T) {
	// Custom small bulkhead for unit test: 2 concurrent, 1 queue slot
	testBulkhead := bulkhead.New(
		bulkhead.WithMaxConcurrent(2),
		bulkhead.WithMaxQueue(1),
		bulkhead.WithQueueTimeout(50*time.Millisecond),
	)
	defer testBulkhead.Close()

	slowHandler := middleware.BulkheadGuard(testBulkhead, "TestService")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	statusCodes := make([]int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			slowHandler.ServeHTTP(rec, req)
			statusCodes[idx] = rec.Code
		}(i)
	}

	wg.Wait()

	rejectedCount := 0
	successCount := 0
	for _, code := range statusCodes {
		if code == http.StatusTooManyRequests {
			rejectedCount++
		} else if code == http.StatusOK {
			successCount++
		}
	}

	if rejectedCount == 0 {
		t.Fatalf("expected at least 1 request to be rejected with 429 Too Many Requests, got status codes: %v", statusCodes)
	}
	if successCount == 0 {
		t.Fatalf("expected at least 1 request to succeed with 200 OK, got status codes: %v", statusCodes)
	}
}
