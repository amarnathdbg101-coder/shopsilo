package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
)

func TestMasterCatalogController(t *testing.T) {
	logger := zap.NewNop()
	mcc := controller.NewMasterCatalogController(nil, logger)

	t.Run("Lookup requires authenticated user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/catalog/lookup/8901058852233", nil)
		w := httptest.NewRecorder()

		mcc.Lookup(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Lookup requires non-empty barcode parameter", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "merchant",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Set empty chi URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("barcode", "   ")
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

		req := httptest.NewRequest(http.MethodGet, "/shops/me/catalog/lookup/", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		mcc.Lookup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for empty barcode, got %d", w.Code)
		}
	})

	t.Run("BulkSeed requires authenticated admin", func(t *testing.T) {
		body, _ := json.Marshal(dto.MasterCatalogBulkSeedRequest{
			Products: []dto.MasterCatalogProductInput{
				{
					Barcode: "8901058852233",
					Name:    "Nestle Maggi 70g",
					MRP:     14.0,
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/admin/catalog/bulk-seed", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mcc.BulkSeed(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for unauthenticated bulk seed, got %d", w.Code)
		}
	})

	t.Run("AdminHardDelete requires authenticated user and valid id", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "admin-1",
			Role:   "admin",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Missing id param
		rctx := chi.NewRouteContext()
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)

		req := httptest.NewRequest(http.MethodDelete, "/admin/products/", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		mcc.AdminHardDeleteProduct(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing id, got %d", w.Code)
		}
	})

	t.Run("CreateMasterProduct rejects invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/catalog/products", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		mcc.CreateMasterProduct(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("UpdateMasterProduct requires valid id parameter", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		req := httptest.NewRequest(http.MethodPut, "/admin/catalog/products/", bytes.NewBufferString("{}")).WithContext(ctx)
		w := httptest.NewRecorder()

		mcc.UpdateMasterProduct(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing id, got %d", w.Code)
		}
	})

	t.Run("RestoreProduct requires valid id parameter", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)

		req := httptest.NewRequest(http.MethodPost, "/admin/products//restore", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		mcc.RestoreProduct(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing id, got %d", w.Code)
		}
	})
}
