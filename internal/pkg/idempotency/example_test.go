package idempotency_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"shopMe/internal/pkg/idempotency"
)

func ExampleEngine_Middleware() {
	idem, _ := idempotency.New(idempotency.WithTTL(5 * time.Minute))
	defer idem.Close()

	var handlerExecutions int32
	handler := idem.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&handlerExecutions, 1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"order_id":"123"}`))
	}))

	// 1. Initial request with idempotency key
	req1 := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req1.Header.Set("X-Idempotency-Key", "order-key-999")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// 2. Duplicate retry with same idempotency key
	req2 := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req2.Header.Set("X-Idempotency-Key", "order-key-999")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	fmt.Printf("Total Handler Executions: %d\n", handlerExecutions)
	fmt.Printf("Second Response Cache Header: %s\n", rec2.Header().Get("X-Cache"))

	// Output:
	// Total Handler Executions: 1
	// Second Response Cache Header: IDEMPOTENCY-HIT
}
