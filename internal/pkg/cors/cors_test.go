package gocors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_Preflight(t *testing.T) {
	cors := New(Config{
		AllowedOrigins: []string{"https://app.shopsilo.com"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         3600,
	})

	target := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("target handler should not be reached during OPTIONS preflight")
	})

	handler := cors.Handler(target)

	req := httptest.NewRequest(http.MethodOptions, "/api/checkout", nil)
	req.Header.Set("Origin", "https://app.shopsilo.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.shopsilo.com" {
		t.Fatalf("unexpected allow origin: %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT" {
		t.Fatalf("unexpected allow methods: %s", rec.Header().Get("Access-Control-Allow-Methods"))
	}
	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Fatalf("unexpected max age: %s", rec.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_ActualRequest(t *testing.T) {
	cors := New(Config{
		AllowedOrigins:   []string{"https://app.shopsilo.com"},
		AllowCredentials: true,
		ExposedHeaders:   []string{"X-Total-Count"},
	})

	targetReached := false
	target := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetReached = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := cors.Handler(target)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("Origin", "https://app.shopsilo.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !targetReached {
		t.Fatal("target handler should have been executed")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.shopsilo.com" {
		t.Fatalf("missing allow origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("missing allow credentials header")
	}
	if rec.Header().Get("Access-Control-Expose-Headers") != "X-Total-Count" {
		t.Fatalf("missing expose headers")
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cors := New(Config{
		AllowedOrigins: []string{"https://app.shopsilo.com"},
	})

	target := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := cors.Handler(target)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Origin header should NOT be present
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disallowed origin should not receive Access-Control-Allow-Origin")
	}
}

func TestCORS_WildcardSubdomain(t *testing.T) {
	cors := New(Config{
		AllowedOrigins: []string{"https://*.shopsilo.com"},
	})

	if !cors.IsOriginAllowed("https://admin.shopsilo.com") {
		t.Fatal("expected admin.shopsilo.com to match wildcard")
	}
	if !cors.IsOriginAllowed("https://seller.shopsilo.com") {
		t.Fatal("expected seller.shopsilo.com to match wildcard")
	}
	if cors.IsOriginAllowed("https://evilshopsilo.com") {
		t.Fatal("evilshopsilo.com should NOT match")
	}
}
