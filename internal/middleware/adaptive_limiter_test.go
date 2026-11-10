package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shopMe/internal/middleware"
)

func TestAdaptiveLimiterMiddleware_AllowsHealthyRequests(t *testing.T) {
	handler := middleware.AdaptiveLimiterMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d with body %s", rec.Code, rec.Body.String())
	}
}
