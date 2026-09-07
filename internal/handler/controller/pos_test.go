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

func TestPOSControllerValidation(t *testing.T) {
	pc := controller.NewPOSController(nil)

	t.Run("CreateSale Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreatePOSSaleRequest{
			PaymentMethod: "cash",
			Items: []dto.POSSaleItemRequest{
				{
					ProductID: "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
					Quantity:  1,
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/sale", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.CreateSale(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("CreateSale Invalid Payment Method", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Email:  "owner@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreatePOSSaleRequest{
			PaymentMethod: "bitcoin", // Only cash, upi, card allowed
			Items: []dto.POSSaleItemRequest{
				{
					ProductID: "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
					Quantity:  1,
				},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/sale", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.CreateSale(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid payment method, got %d", w.Code)
		}
	})

	t.Run("CreateSale Empty Items", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Email:  "owner@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreatePOSSaleRequest{
			PaymentMethod: "cash",
			Items:         []dto.POSSaleItemRequest{},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/sale", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.CreateSale(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for empty items, got %d", w.Code)
		}
	})

	t.Run("GetDailySummary Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/daily-summary", nil)
		w := httptest.NewRecorder()

		pc.GetDailySummary(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DownloadReceiptPDF Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/receipts/BIL-001.pdf", nil)
		w := httptest.NewRecorder()

		pc.DownloadReceiptPDF(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ViewPublicReceiptPDF Missing Bill Number", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/receipts/", nil)
		w := httptest.NewRecorder()

		pc.ViewPublicReceiptPDF(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing bill number, got %d", w.Code)
		}
	})
}
