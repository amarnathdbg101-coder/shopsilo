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

func TestKhataControllerValidation(t *testing.T) {
	kc := controller.NewKhataController(nil)

	t.Run("RecordCredit Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.RecordCreditTransactionRequest{
			CustomerName:   "Ramesh Kumar",
			CustomerMobile: "9876543210",
			Amount:         500.0,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/khata", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		kc.RecordCredit(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("RecordCredit Invalid Body - Negative Amount", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.RecordCreditTransactionRequest{
			CustomerName:   "Ramesh Kumar",
			CustomerMobile: "9876543210",
			Amount:         -50.0,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/khata", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		kc.RecordCredit(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("RecordCredit Invalid Body - Missing Mobile", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.RecordCreditTransactionRequest{
			CustomerName: "Ramesh Kumar",
			Amount:       150.0,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/khata", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		kc.RecordCredit(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("RecordPayment Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.RecordPaymentRequest{
			Amount:      200.0,
			PaymentMode: "cash",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/khata/9876543210/payment", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		kc.RecordPayment(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("RecordPayment Invalid Payment Mode", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.RecordPaymentRequest{
			Amount:      200.0,
			PaymentMode: "crypto", // invalid
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/khata/9876543210/payment", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		kc.RecordPayment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ListCustomers Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata", nil)
		w := httptest.NewRecorder()

		kc.ListCustomers(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetSummary Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata/summary", nil)
		w := httptest.NewRecorder()

		kc.GetSummary(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetPaymentReminder Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata/9876543210/reminder", nil)
		w := httptest.NewRecorder()

		kc.GetPaymentReminder(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("UpdateCreditLimit Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.SetCreditLimitRequest{CreditLimit: 5000})
		req := httptest.NewRequest(http.MethodPut, "/shops/me/khata/9876543210/credit-limit", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		kc.UpdateCreditLimit(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("UpdateCreditLimit Invalid Body - Negative Limit", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.SetCreditLimitRequest{CreditLimit: -100})
		req := httptest.NewRequest(http.MethodPut, "/shops/me/khata/9876543210/credit-limit", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		kc.UpdateCreditLimit(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetAgingReport Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata/aging", nil)
		w := httptest.NewRecorder()

		kc.GetAgingReport(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DownloadStatementPDF Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata/9876543210/statement.pdf", nil)
		w := httptest.NewRecorder()

		kc.DownloadStatementPDF(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetStatementShare Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/khata/9876543210/statement/share", nil)
		w := httptest.NewRecorder()

		kc.GetStatementShare(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
