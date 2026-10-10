package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBotGuard_LegitimateTraffic(t *testing.T) {
	ClearIPJail("192.168.1.50")

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Legitimate Chrome Browser
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.RemoteAddr = "192.168.1.50:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for legitimate browser, got %d", w.Code)
	}

	// Legitimate Flutter Mobile App
	reqMobile := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	reqMobile.Host = "api.shopsilo.in"
	reqMobile.Header.Set("User-Agent", "ShopSilo/1.0.0 (Flutter; Android 14)")
	reqMobile.RemoteAddr = "192.168.1.50:1234"
	wMobile := httptest.NewRecorder()
	handler.ServeHTTP(wMobile, reqMobile)

	if wMobile.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for Flutter mobile app, got %d", wMobile.Code)
	}
}

func TestBotGuard_WAF_SQLInjection(t *testing.T) {
	testIP := "192.168.1.55"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Attack with UNION SELECT
	req := httptest.NewRequest(http.MethodGet, "/products?search=shoes'+UNION+SELECT+1,2,3--", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = testIP + ":5432"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for SQL Injection query, got %d", w.Code)
	}

	// IP must be quarantined by WAF
	jailed, _, _ := IsIPJailed(testIP)
	if !jailed {
		t.Fatalf("expected IP %s to be jailed after SQL injection attempt", testIP)
	}
}

func TestBotGuard_WAF_PathTraversal(t *testing.T) {
	testIP := "192.168.1.56"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalog?file=../../etc/passwd", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = testIP + ":5432"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Path Traversal attempt, got %d", w.Code)
	}
}

func TestBotGuard_WAF_XSS(t *testing.T) {
	testIP := "192.168.1.57"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products?q=<script>alert(1)</script>", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = testIP + ":5432"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for XSS probe, got %d", w.Code)
	}
}

func TestBotGuard_DisallowedMethod(t *testing.T) {
	testIP := "192.168.1.58"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("TRACE", "/products", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = testIP + ":5432"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed for TRACE, got %d", w.Code)
	}
}

func TestBotGuard_MissingHost(t *testing.T) {
	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Host = ""
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "192.168.1.59:5432"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing Host header, got %d", w.Code)
	}
}

func TestBodySizeGuard(t *testing.T) {
	// Guard with 100 bytes max limit
	guard := BodySizeGuard(100)
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 200)
		_, err := r.Body.Read(buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Payload with 50 bytes -> OK
	smallBody := bytes.Repeat([]byte("A"), 50)
	reqSmall := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(smallBody))
	reqSmall.ContentLength = 50
	wSmall := httptest.NewRecorder()
	handler.ServeHTTP(wSmall, reqSmall)

	if wSmall.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for small payload, got %d", wSmall.Code)
	}

	// Payload with 500 bytes -> 413 Request Entity Too Large
	largeBody := bytes.Repeat([]byte("B"), 500)
	reqLarge := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(largeBody))
	reqLarge.ContentLength = 500
	wLarge := httptest.NewRecorder()
	handler.ServeHTTP(wLarge, reqLarge)

	if wLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 Request Entity Too Large for 500 bytes payload, got %d", wLarge.Code)
	}
}

func TestBotGuard_ExploitProbe(t *testing.T) {
	testIP := "192.168.1.60"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	probes := []string{"/.env", "/wp-admin", "/xmlrpc.php", "/phpmyadmin", "/cgi-bin/test"}
	for _, probe := range probes {
		req := httptest.NewRequest(http.MethodGet, probe, nil)
		req.Host = "api.shopsilo.in"
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.RemoteAddr = testIP + ":5555"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for probe %s, got %d", probe, w.Code)
		}
	}

	// Verify that IP is now jailed
	jailed, _, _ := IsIPJailed(testIP)
	if !jailed {
		t.Fatalf("expected IP %s to be jailed after probe attempts", testIP)
	}
}

func TestBotGuard_MaliciousScanner(t *testing.T) {
	testIP := "192.168.1.70"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "sqlmap/1.5.2#stable (http://sqlmap.org)")
	req.RemoteAddr = testIP + ":4444"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for sqlmap scanner, got %d", w.Code)
	}

	// IP must be quarantined
	jailed, _, _ := IsIPJailed(testIP)
	if !jailed {
		t.Fatalf("expected IP %s to be jailed after scanner activity", testIP)
	}
}

func TestBotGuard_AutomatedScraperOnPost(t *testing.T) {
	testIP := "192.168.1.80"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "python-requests/2.28.1")
	req.RemoteAddr = testIP + ":3333"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for python-requests on POST, got %d", w.Code)
	}
}

func TestBotGuard_EmptyUAOnPost(t *testing.T) {
	testIP := "192.168.1.90"
	ClearIPJail(testIP)

	handler := BotGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.Host = "api.shopsilo.in"
	req.Header.Set("User-Agent", "")
	req.RemoteAddr = testIP + ":2222"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty User-Agent on POST, got %d", w.Code)
	}
}

func TestValidateHoneypot(t *testing.T) {
	testIP := "192.168.1.95"
	ClearIPJail(testIP)

	// Valid submission (empty honeypot)
	if !ValidateHoneypot("", testIP) {
		t.Fatalf("expected empty honeypot to be valid")
	}

	// Bot filled honeypot
	if ValidateHoneypot("spam bot text", testIP) {
		t.Fatalf("expected non-empty honeypot to fail validation")
	}

	// IP should now be jailed
	jailed, _, _ := IsIPJailed(testIP)
	if !jailed {
		t.Fatalf("expected IP %s to be jailed after triggering honeypot", testIP)
	}
}

func TestJailPurge(t *testing.T) {
	testIP := "192.168.1.99"
	JailIP(testIP, 10*time.Millisecond, "temporary test jail")

	jailed, _, _ := IsIPJailed(testIP)
	if !jailed {
		t.Fatalf("expected IP to be jailed")
	}

	time.Sleep(20 * time.Millisecond)
	PurgeExpiredJail()

	jailedAfter, _, _ := IsIPJailed(testIP)
	if jailedAfter {
		t.Fatalf("expected IP to be released after expiry and purge")
	}
}
