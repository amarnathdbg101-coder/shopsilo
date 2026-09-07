package controller_test

import (
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"testing"
)

func TestAnalyticsControllerValidation(t *testing.T) {
	ac := controller.NewAnalyticsController(nil)

	t.Run("GetMonthlyProfit Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/analytics/profit?year=2026&month=9", nil)
		w := httptest.NewRecorder()

		ac.GetMonthlyProfit(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetProductMatrix Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/analytics/products", nil)
		w := httptest.NewRecorder()

		ac.GetProductMatrix(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
