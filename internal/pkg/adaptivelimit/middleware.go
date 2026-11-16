package adaptivelimit

import "net/http"

// Middleware wraps an http.Handler with adaptive concurrency limiting.
func Middleware(limiter *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			release, err := limiter.Acquire(r.Context())
			if err != nil {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"system_overloaded","message":"adaptive concurrency limit exceeded"}`))
				return
			}
			defer release()

			next.ServeHTTP(w, r)
		})
	}
}
