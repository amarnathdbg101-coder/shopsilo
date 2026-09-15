package middleware

import (
	"net"
	"net/http"
	"shopMe/internal/reuse"
	"strconv"
	"strings"
	"sync"
	"time"
)

type clientRecord struct {
	count     int
	windowEnd time.Time
}

// IPRateLimiter tracks request counts per IP in memory
type IPRateLimiter struct {
	mu          sync.RWMutex
	records     map[string]*clientRecord
	limit       int
	window      time.Duration
	stopCleanup chan struct{}
}

// NewIPRateLimiter creates a new rate limiter with background garbage collection
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		records:     make(map[string]*clientRecord),
		limit:       limit,
		window:      window,
		stopCleanup: make(chan struct{}),
	}

	// Background routine to prune expired client records and prevent memory leak
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				limiter.cleanup()
			case <-limiter.stopCleanup:
				return
			}
		}
	}()

	return limiter
}

func (l *IPRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, rec := range l.records {
		if now.After(rec.windowEnd) {
			delete(l.records, ip)
		}
	}
}

// Stop terminates the background cleanup routine
func (l *IPRateLimiter) Stop() {
	close(l.stopCleanup)
}

// CheckAndAllow checks if the given IP is within the rate limit and returns headers metadata
func (l *IPRateLimiter) CheckAndAllow(ip string) (allowed bool, remaining int, resetInSec int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	rec, exists := l.records[ip]

	if !exists || now.After(rec.windowEnd) {
		rec = &clientRecord{
			count:     1,
			windowEnd: now.Add(l.window),
		}
		l.records[ip] = rec
		remaining = l.limit - 1
		resetInSec = int64(time.Until(rec.windowEnd).Seconds())
		if resetInSec < 0 {
			resetInSec = 0
		}
		return true, remaining, resetInSec
	}

	resetInSec = int64(time.Until(rec.windowEnd).Seconds())
	if resetInSec < 0 {
		resetInSec = 0
	}

	if rec.count < l.limit {
		rec.count++
		remaining = l.limit - rec.count
		return true, remaining, resetInSec
	}

	remaining = 0
	return false, remaining, resetInSec
}

// Allow checks if the given IP is within the rate limit
func (l *IPRateLimiter) Allow(ip string) bool {
	allowed, _, _ := l.CheckAndAllow(ip)
	return allowed
}

// ExtractIP extracts the client IP address from request headers or remote addr
func ExtractIP(r *http.Request) string {
	// 1. Check Cloudflare header
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}

	// 2. Check X-Forwarded-For header (first entry is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}

	// 3. Check X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// 4. Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// Middleware returns a Chi middleware handler function
func (l *IPRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ExtractIP(r)

			allowed, remaining, resetInSec := l.CheckAndAllow(ip)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetInSec, 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(resetInSec, 10))
				reuse.Error(w, http.StatusTooManyRequests, "Too many requests. Please slow down and try again in a moment.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Pre-configured standard limiters
var (
	// AuthRateLimiter limits sensitive login/register/reset requests (10 requests per minute)
	AuthRateLimiter = NewIPRateLimiter(10, 1*time.Minute)

	// UploadRateLimiter limits heavy image uploads to Cloudflare R2 (20 uploads per minute)
	UploadRateLimiter = NewIPRateLimiter(20, 1*time.Minute)

	// GlobalRateLimiter protects the API from aggressive scraping or DoS (120 requests per minute)
	GlobalRateLimiter = NewIPRateLimiter(120, 1*time.Minute)
)
