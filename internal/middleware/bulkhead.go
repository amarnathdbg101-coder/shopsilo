// Package middleware provides HTTP interceptors, rate limiters, and resilience controls.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/amarnathdbg101-coder/go/go-bulkhead"
)

var (
	// AIBulkhead isolates AI Vision OCR, Voice Bill, and Studio processing to prevent thread exhaustion.
	AIBulkhead = bulkhead.New(
		bulkhead.WithMaxConcurrent(8),
		bulkhead.WithMaxQueue(20),
		bulkhead.WithQueueTimeout(3*time.Second),
	)

	// PDFBulkhead isolates thermal receipt, standee, and EOD P&L PDF rendering engines.
	PDFBulkhead = bulkhead.New(
		bulkhead.WithMaxConcurrent(6),
		bulkhead.WithMaxQueue(15),
		bulkhead.WithQueueTimeout(4*time.Second),
	)
)

// BulkheadGuard wraps an HTTP handler inside a Netflix/Hystrix-style concurrency bulkhead.
// If the concurrent execution slots and queue are full, it fails fast with HTTP 429 Too Many Requests,
// preventing slow CPU-heavy requests from crashing the database connection pool or starving POS checkout.
func BulkheadGuard(engine *bulkhead.Bulkhead, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := engine.Execute(r.Context(), func() error {
				next.ServeHTTP(w, r)
				return nil
			})

			if err != nil {
				if errors.Is(err, bulkhead.ErrBulkheadFull) || errors.Is(err, bulkhead.ErrQueueTimeout) {
					w.Header().Set("Retry-After", "2")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = fmt.Fprintf(w, `{"success":false,"error":"%s is under high load. Please retry in a few seconds."}`+"\n", serviceName)
					return
				}
			}
		})
	}
}
