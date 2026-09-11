package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/middleware"
	"testing"
)

func TestExtractClientIP(t *testing.T) {
	t.Run("Extract from X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")

		ip := middleware.ExtractClientIP(req)
		if ip != "203.0.113.195" {
			t.Fatalf("expected 203.0.113.195, got %s", ip)
		}
	})

	t.Run("Extract from X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "198.51.100.1")

		ip := middleware.ExtractClientIP(req)
		if ip != "198.51.100.1" {
			t.Fatalf("expected 198.51.100.1, got %s", ip)
		}
	})

	t.Run("Fallback to RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:54321"

		ip := middleware.ExtractClientIP(req)
		if ip != "192.0.2.1" {
			t.Fatalf("expected 192.0.2.1, got %s", ip)
		}
	})
}

func TestRequireRole(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	t.Run("Denies access if not admin", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-1",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		req := httptest.NewRequest(http.MethodGet, "/admin/reports", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		middleware.RequireRole("admin")(nextHandler).ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("Allows access if admin", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "admin-1",
			Role:   "admin",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		req := httptest.NewRequest(http.MethodGet, "/admin/reports", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		middleware.RequireRole("admin")(nextHandler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})
}

func TestInvalidateBanCache(t *testing.T) {
	middleware.InvalidateBanCache()
}

