package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
	"testing"
)

func TestInventoryControllerValidation(t *testing.T) {
	ic := controller.NewInventoryController(nil)

	t.Run("AdjustStock Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.AdjustStockRequest{
			ProductID:  "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
			Adjustment: 50,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/inventory/adjust", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		ic.AdjustStock(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("AdjustStock Invalid Product UUID", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Email:  "owner@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.AdjustStockRequest{
			ProductID:  "not-a-uuid",
			Adjustment: 50,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/inventory/adjust", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		ic.AdjustStock(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetLowStockAlerts Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/inventory/low-stock", nil)
		w := httptest.NewRecorder()

		ic.GetLowStockAlerts(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DownloadReorderSheetPDF Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/inventory/reorder-sheet.pdf", nil)
		w := httptest.NewRecorder()

		ic.DownloadReorderSheetPDF(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
