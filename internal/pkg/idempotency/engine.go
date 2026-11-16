package idempotency

import (
	"bytes"
	"net/http"
)

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

// Engine coordinates request inspection, deduplication locking, and cached response replay.
type Engine struct {
	cfg *Config
}

// NewEngine creates an Idempotency engine with the given options.
func NewEngine(opts ...Option) (*Engine, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Engine{cfg: cfg}, nil
}

// Close closes the underlying storage driver.
func (e *Engine) Close() error {
	if e.cfg != nil && e.cfg.Storage != nil {
		return e.cfg.Storage.Close()
	}
	return nil
}

// Middleware returns a standard net/http middleware handler.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only mutating HTTP methods require idempotency guards
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(e.cfg.KeyHeader)
		if key == "" && e.cfg.AltKeyHeader != "" {
			key = r.Header.Get(e.cfg.AltKeyHeader)
		}

		// If no idempotency key is present, proceed normally
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		storageKey := e.cfg.KeyGenerator(r, key)

		// 1. Check if response is already cached
		cached, err := e.cfg.Storage.Get(ctx, storageKey)
		if err != nil && e.cfg.Hooks.OnError != nil {
			e.cfg.Hooks.OnError(storageKey, err)
		}
		if err == nil && cached != nil {
			for k, values := range cached.Headers {
				for _, val := range values {
					w.Header().Add(k, val)
				}
			}
			w.Header().Set("X-Cache", "IDEMPOTENCY-HIT")
			w.WriteHeader(cached.StatusCode)
			_, _ = w.Write(cached.Body)
			if e.cfg.Hooks.OnHit != nil {
				e.cfg.Hooks.OnHit(storageKey, cached.StatusCode)
			}
			return
		}

		// 2. Acquire in-flight lock to prevent duplicate concurrent execution
		acquired, err := e.cfg.Storage.Lock(ctx, storageKey, e.cfg.LockTTL)
		if err != nil || !acquired {
			if e.cfg.Hooks.OnConflict != nil {
				e.cfg.Hooks.OnConflict(storageKey)
			}
			w.Header().Set("Retry-After", "1")
			http.Error(w, ErrKeyInFlight.Error(), http.StatusConflict)
			return
		}

		// Ensure lock is released if request panics or fails before completing
		defer func() {
			_ = e.cfg.Storage.Unlock(ctx, storageKey)
		}()

		// 3. Record response during execution
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		// 4. Cache response if qualifying status code
		if e.cfg.ShouldCache(rec.statusCode) {
			recRecord := &Record{
				StatusCode: rec.statusCode,
				Headers:    rec.Header().Clone(),
				Body:       rec.body.Bytes(),
			}
			_ = e.cfg.Storage.Set(ctx, storageKey, recRecord, e.cfg.TTL)
			if e.cfg.Hooks.OnSaved != nil {
				e.cfg.Hooks.OnSaved(storageKey, rec.statusCode)
			}
		}
	})
}
