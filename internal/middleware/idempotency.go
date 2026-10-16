// Package middleware provides HTTP interceptors.
package middleware

import (
	"bytes"
	"net/http"
	"shopMe/internal/reuse"
	"strings"
	"sync"
	"time"
)

type idempotencyStatus string

const (
	statusProcessing idempotencyStatus = "PROCESSING"
	statusCompleted  idempotencyStatus = "COMPLETED"
)

type idempotencyRecord struct {
	status     idempotencyStatus
	statusCode int
	body       []byte
	headers    http.Header
	expiresAt  time.Time
}

var (
	idempotencyMu    sync.RWMutex
	idempotencyStore = make(map[string]*idempotencyRecord)
)

func init() {
	// Background garbage collection routine to purge expired keys and prevent memory leaks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			idempotencyMu.Lock()
			now := time.Now()
			for k, v := range idempotencyStore {
				if now.After(v.expiresAt) {
					delete(idempotencyStore, k)
				}
			}
			idempotencyMu.Unlock()
		}
	}()
}

// recordingResponseWriter wraps http.ResponseWriter to capture the status code and output body for idempotency caching
type recordingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *recordingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *recordingResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Idempotency intercepts mutating POST/PUT requests and replays cached responses for identical idempotency keys
func Idempotency(ttl ...time.Duration) func(http.Handler) http.Handler {
	recordTTL := 10 * time.Minute
	if len(ttl) > 0 && ttl[0] > 0 {
		recordTTL = ttl[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract idempotency key from standard headers
			key := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
			if key == "" {
				key = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			}

			// If no idempotency key is provided, execute normally
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if key already exists in store
			idempotencyMu.Lock()
			record, exists := idempotencyStore[key]
			now := time.Now()

			if exists && now.Before(record.expiresAt) {
				if record.status == statusProcessing {
					idempotencyMu.Unlock()
					reuse.Error(w, http.StatusConflict, "A request with this idempotency key is already currently processing. Please wait.")
					return
				}

				if record.status == statusCompleted {
					idempotencyMu.Unlock()
					// Replay cached headers
					for k, vv := range record.headers {
						for _, v := range vv {
							w.Header().Add(k, v)
						}
					}
					w.Header().Set("X-Cache-Lookup", "HIT-IDEMPOTENT")
					w.WriteHeader(record.statusCode)
					_, _ = w.Write(record.body)
					return
				}
			}

			// Mark key as PROCESSING
			idempotencyStore[key] = &idempotencyRecord{
				status:    statusProcessing,
				expiresAt: now.Add(recordTTL),
			}
			idempotencyMu.Unlock()

			// Wrap response writer to record output
			rec := &recordingResponseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}

			next.ServeHTTP(rec, r)

			// Handler finished: if successful, cache response; otherwise clear key so client can retry
			idempotencyMu.Lock()
			defer idempotencyMu.Unlock()

			if rec.statusCode >= 200 && rec.statusCode < 300 {
				idempotencyStore[key] = &idempotencyRecord{
					status:     statusCompleted,
					statusCode: rec.statusCode,
					body:       rec.body.Bytes(),
					headers:    rec.Header().Clone(),
					expiresAt:  time.Now().Add(recordTTL),
				}
			} else {
				delete(idempotencyStore, key)
			}
		})
	}
}
