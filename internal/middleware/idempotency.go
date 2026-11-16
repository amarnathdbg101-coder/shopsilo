package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type cachedResponse struct {
	statusCode int
	body       []byte
	headers    http.Header
	expiresAt  time.Time
}

var (
	idempotencyMu    sync.RWMutex
	idempotencyStore = make(map[string]cachedResponse)
)

func init() {
	// Background garbage collector for expired idempotency records
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
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

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// IdempotencyGuard catches X-Idempotency-Key headers on mutating requests and returns cached responses for network retries.
func IdempotencyGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only inspect mutating HTTP methods (POST, PUT, PATCH, DELETE)
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-Idempotency-Key")
		if key == "" {
			key = r.Header.Get("Idempotency-Key")
		}

		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Composite key = IP + Method + Path + IdempotencyKey
		ip := ExtractIP(r)
		compositeKey := ip + ":" + r.Method + ":" + r.URL.Path + ":" + key

		// 1. Check if response is already cached for this idempotency key
		idempotencyMu.RLock()
		cached, exists := idempotencyStore[compositeKey]
		idempotencyMu.RUnlock()

		if exists && time.Now().Before(cached.expiresAt) {
			for k, values := range cached.headers {
				for _, val := range values {
					w.Header().Add(k, val)
				}
			}
			w.Header().Set("X-Cache", "IDEMPOTENCY-HIT")
			w.WriteHeader(cached.statusCode)
			_, _ = w.Write(cached.body)
			return
		}

		// 2. Intercept and record the response
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		// 3. Cache successful or client-error responses for 60 seconds
		if rec.statusCode < 500 {
			idempotencyMu.Lock()
			idempotencyStore[compositeKey] = cachedResponse{
				statusCode: rec.statusCode,
				body:       rec.body.Bytes(),
				headers:    rec.Header().Clone(),
				expiresAt:  time.Now().Add(60 * time.Second),
			}
			idempotencyMu.Unlock()
		}
	})
}
