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

	t.Run("ScanBarcode Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/scan/WM-001", nil)
		w := httptest.NewRecorder()

		pc.ScanBarcode(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ShareBill Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/receipts/BIL-001/share", nil)
		w := httptest.NewRecorder()

		pc.ShareBill(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetDailyCloseReport Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/day-close", nil)
		w := httptest.NewRecorder()

		pc.GetDailyCloseReport(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetMonthlyGSTReport Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/gst-report?year=2026&month=9", nil)
		w := httptest.NewRecorder()

		pc.GetMonthlyGSTReport(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ParkBill Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.ParkPOSBillRequest{
			Label: "Customer in red shirt",
			Items: []dto.POSSaleItemRequest{
				{ProductID: "b9c02052-a5e2-45a7-96a9-e85d45db60ea", Quantity: 2},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/park", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.ParkBill(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ParkBill Empty Items", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.ParkPOSBillRequest{
			Label: "Customer in red shirt",
			Items: []dto.POSSaleItemRequest{},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/park", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		pc.ParkBill(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ListParkedBills Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/park", nil)
		w := httptest.NewRecorder()

		pc.ListParkedBills(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetParkedBill Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/park/p123", nil)
		w := httptest.NewRecorder()

		pc.GetParkedBill(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DeleteParkedBill Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/shops/me/pos/park/p123", nil)
		w := httptest.NewRecorder()

		pc.DeleteParkedBill(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ParseParchi Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.ParseParchiRequest{
			RawText: "2kg basmati chawal\n500g toor dal",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/parse-parchi", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.ParseParchi(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ParseParchi Invalid Body", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Empty raw_text (min 2 required)
		body, _ := json.Marshal(dto.ParseParchiRequest{
			RawText: "",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/parse-parchi", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.ParseParchi(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetWeeklyScorecard Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/weekly-scorecard", nil)
		w := httptest.NewRecorder()

		pc.GetWeeklyScorecard(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("CreateSale Split Payment with missing split details", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreatePOSSaleRequest{
			PaymentMethod: "split",
			Items: []dto.POSSaleItemRequest{
				{ProductID: "a0000000-0000-0000-0000-000000000001", Quantity: 1},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/sale", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.CreateSale(w, req)

		// Should either fail with bad request or service error
		if w.Code == http.StatusOK {
			t.Fatalf("expected error response when split payments details are missing")
		}
	})

	t.Run("GetCustomerRecentBasket Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/customers/9876543210/recent-basket", nil)
		w := httptest.NewRecorder()

		pc.GetCustomerRecentBasket(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ProcessPOSReturn Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.ProcessPOSReturnRequest{
			BillNumber: "POS-12345",
			Items: []dto.POSReturnItemRequest{
				{ProductID: "a0000000-0000-0000-0000-000000000001", Quantity: 1},
			},
			RefundMode: "cash",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/returns", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.ProcessPOSReturn(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ProcessPOSReturn Invalid Body", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Invalid: missing BillNumber and invalid refund mode
		body, _ := json.Marshal(dto.ProcessPOSReturnRequest{
			BillNumber: "",
			RefundMode: "bitcoin",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/pos/returns", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.ProcessPOSReturn(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}
