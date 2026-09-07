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

func TestLoyaltyControllerValidation(t *testing.T) {
	lc := controller.NewLoyaltyController(nil)

	t.Run("ProcessReturn Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReturnRequest{
			ProductID:    "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
			Quantity:     1,
			RefundAmount: 500,
			Reason:       "Defective item",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/returns", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		lc.ProcessReturn(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ProcessReturn Invalid UUID", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Email:  "owner@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreateReturnRequest{
			ProductID:    "invalid-uuid",
			Quantity:     1,
			RefundAmount: 500,
			Reason:       "Defective item",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/returns", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		lc.ProcessReturn(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid UUID, got %d", w.Code)
		}
	})

	t.Run("CreateOffer Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateOfferRequest{
			Title:        "Weekend Special",
			DiscountText: "Flat 20% Off",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/offers", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		lc.CreateOffer(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ListOffers Missing Slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops//offers", nil)
		w := httptest.NewRecorder()

		lc.ListOffers(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing slug, got %d", w.Code)
		}
	})

	t.Run("GetUserLoyalty Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/user/loyalty", nil)
		w := httptest.NewRecorder()

		lc.GetUserLoyalty(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ScanProduct Missing Code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/scan/", nil)
		w := httptest.NewRecorder()

		lc.ScanProduct(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing barcode, got %d", w.Code)
		}
	})
}
