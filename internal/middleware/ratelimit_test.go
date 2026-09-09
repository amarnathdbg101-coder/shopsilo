package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewIPRateLimiter(3, 100*time.Millisecond)
	defer limiter.Stop()

	testIP := "192.168.1.100"

	// 1st to 3rd requests should be allowed
	for i := 1; i <= 3; i++ {
		if !limiter.Allow(testIP) {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	// 4th request within window should be rejected
	if limiter.Allow(testIP) {
		t.Fatalf("4th request should exceed rate limit")
	}

	// Wait for window to slide/expire
	time.Sleep(120 * time.Millisecond)

	// After expiry, request should be allowed again
	if !limiter.Allow(testIP) {
		t.Fatalf("request after window expiry should be allowed")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	limiter := NewIPRateLimiter(2, 500*time.Millisecond)
	defer limiter.Stop()

	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// First two requests should return 200 OK
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.5:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 on request %d, got %d", i, w.Code)
		}
	}

	// Third request should be blocked with 429
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 Too Many Requests, got %d", w.Code)
	}
}

func TestExtractIP(t *testing.T) {
	// Case 1: CF-Connecting-IP
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req1.Header.Set("X-Forwarded-For", "198.51.100.1")
	if ip := ExtractIP(req1); ip != "203.0.113.1" {
		t.Errorf("expected CF-Connecting-IP 203.0.113.1, got %s", ip)
	}

	// Case 2: X-Forwarded-For
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	if ip := ExtractIP(req2); ip != "198.51.100.1" {
		t.Errorf("expected client IP 198.51.100.1, got %s", ip)
	}

	// Case 3: RemoteAddr fallback
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "172.16.0.10:54321"
	if ip := ExtractIP(req3); ip != "172.16.0.10" {
		t.Errorf("expected RemoteAddr IP 172.16.0.10, got %s", ip)
	}
}
