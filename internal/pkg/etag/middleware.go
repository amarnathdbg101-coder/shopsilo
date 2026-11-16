package goetag

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
)

type middlewareConfig struct {
	mode         Mode
	cacheControl string
	skipPaths    []string
}

// Option configures ETag HTTP middleware behavior.
type Option func(*middlewareConfig)

// WithWeakETag configures middleware to emit lightweight weak ETags (W/"...").
func WithWeakETag() Option {
	return func(c *middlewareConfig) {
		c.mode = ModeWeak
	}
}

// WithCacheControl sets custom Cache-Control header directive (default: "private, no-cache, must-revalidate").
func WithCacheControl(directive string) Option {
	return func(c *middlewareConfig) {
		if directive != "" {
			c.cacheControl = directive
		}
	}
}

// WithSkipPaths excludes designated URL paths from automatic ETag calculation.
func WithSkipPaths(paths ...string) Option {
	return func(c *middlewareConfig) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type etagResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
	wroteHead  bool
}

func (w *etagResponseWriter) WriteHeader(code int) {
	if !w.wroteHead {
		w.statusCode = code
		w.wroteHead = true
	}
}

func (w *etagResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHead {
		w.WriteHeader(http.StatusOK)
	}
	return w.buf.Write(b)
}

// Middleware creates an automatic HTTP ETag interceptor.
// Automatically calculates ETags for successful (200 OK) GET/HEAD responses,
// sets Cache-Control and ETag headers, and responds with HTTP 304 Not Modified when fresh.
func Middleware(opts ...Option) func(http.Handler) http.Handler {
	cfg := middlewareConfig{
		mode:         ModeStrong,
		cacheControl: "private, no-cache, must-revalidate",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET and HEAD requests
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			// Check skip paths
			for _, skip := range cfg.skipPaths {
				if strings.HasPrefix(r.URL.Path, skip) {
					next.ServeHTTP(w, r)
					return
				}
			}

			buf := bufPool.Get().(*bytes.Buffer)
			buf.Reset()
			defer bufPool.Put(buf)

			rw := &etagResponseWriter{
				ResponseWriter: w,
				buf:            buf,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			// Only process 200 OK responses
			if rw.statusCode != http.StatusOK {
				w.WriteHeader(rw.statusCode)
				_, _ = w.Write(buf.Bytes())
				return
			}

			bodyBytes := buf.Bytes()
			var tag string
			if cfg.mode == ModeWeak {
				tag = GenerateWeak(bodyBytes)
			} else {
				tag = Generate(bodyBytes)
			}

			w.Header().Set("ETag", tag)
			w.Header().Set("Cache-Control", cfg.cacheControl)

			// Check If-None-Match
			if clientETags := r.Header.Get("If-None-Match"); clientETags != "" {
				if Matches(clientETags, tag, true) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}

			w.WriteHeader(rw.statusCode)
			if r.Method != http.MethodHead {
				_, _ = w.Write(bodyBytes)
			}
		})
	}
}
