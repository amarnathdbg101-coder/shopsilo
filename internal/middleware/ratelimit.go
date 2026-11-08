package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"shopMe/internal/reuse"

	"github.com/amarnathdbg101-coder/go/go-slidingwindow"
)

// IPRateLimiter provides dual-tier sliding-window rate limiting with micro-burst surge defense.
// Powered by zero-dep go-slidingwindow ring buffers.
type IPRateLimiter struct {
	mu           sync.Mutex
	window       time.Duration
	limit        int
	burstLimit   int
	mainLimiter  *slidingwindow.KeyedLimiter
	burstLimiter *slidingwindow.KeyedLimiter
	violations   map[string]int
}

// NewIPRateLimiter creates an atomic sliding-window rate limiter with sub-second burst protection.
func NewIPRateLimiter(limit int, burstLimit int, window time.Duration) *IPRateLimiter {
	// Granular ring-buffer buckets across the window (min 10 buckets for precision)
	bucketCount := 20
	if window < 200*time.Millisecond {
		bucketCount = 10
	}
	main, _ := slidingwindow.NewKeyedLimiter(window, bucketCount, int64(limit))

	var burst *slidingwindow.KeyedLimiter
	if burstLimit > 0 {
		// 2-second sub-window for micro-burst protection
		burst, _ = slidingwindow.NewKeyedLimiter(2*time.Second, 10, int64(burstLimit))
	}

	return &IPRateLimiter{
		window:       window,
		limit:        limit,
		burstLimit:   burstLimit,
		mainLimiter:  main,
		burstLimiter: burst,
		violations:   make(map[string]int),
	}
}

// Stop terminates rate limiter resources (sliding window ring buffers require no background threads).
func (l *IPRateLimiter) Stop() {}

// CheckAndAllow checks if the given IP is within the rate limit and burst quota.
func (l *IPRateLimiter) CheckAndAllow(ip string) (allowed bool, remaining int, resetInSec int64) {
	resetInSec = int64(l.window.Seconds())
	if resetInSec < 1 {
		resetInSec = 1
	}

	// 1. Micro-burst check (2-second sub-window surge limiter)
	if l.burstLimiter != nil && !l.burstLimiter.Allow(ip) {
		return false, 0, resetInSec
	}

	// 2. Main rolling sliding-window quota check
	if !l.mainLimiter.Allow(ip) {
		l.mu.Lock()
		l.violations[ip]++
		if l.violations[ip] >= 4 {
			// Escalate repeat abusers to IP Jail for 15 minutes
			JailIP(ip, 15*time.Minute, "excessive rate limit violations")
		}
		l.mu.Unlock()
		return false, 0, resetInSec
	}

	currentCount := int(l.mainLimiter.Count(ip))
	remaining = l.limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, resetInSec
}

// Allow checks if the given IP is within the rate limit.
func (l *IPRateLimiter) Allow(ip string) bool {
	allowed, _, _ := l.CheckAndAllow(ip)
	return allowed
}

// ExtractIP extracts the client IP address from request headers or remote addr.
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

// Middleware returns a Chi/net/http middleware handler function.
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
	// AuthRateLimiter limits sensitive login/register/reset requests (15 requests/min, max 5 per 2s burst)
	AuthRateLimiter = NewIPRateLimiter(15, 5, 1*time.Minute)

	// OTPRateLimiter restricts rapid OTP generation requests (4 requests/10min, max 2 per 2s burst)
	OTPRateLimiter = NewIPRateLimiter(4, 2, 10*time.Minute)

	// MutationRateLimiter protects database writes like placing orders or product mutations (30/min, max 8 per 2s burst)
	MutationRateLimiter = NewIPRateLimiter(30, 8, 1*time.Minute)

	// UploadRateLimiter limits heavy image uploads to Cloudflare R2 (60 uploads/min, max 20 per 2s burst)
	UploadRateLimiter = NewIPRateLimiter(60, 20, 1*time.Minute)

	// GlobalRateLimiter protects the API from aggressive scraping or DoS (600 requests/min, max 120 per 2s burst)
	GlobalRateLimiter = NewIPRateLimiter(600, 120, 1*time.Minute)
)
