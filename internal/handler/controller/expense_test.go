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

func TestExpenseControllerValidation(t *testing.T) {
	ec := controller.NewExpenseController(nil)

	t.Run("CreateExpense Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateExpenseRequest{
			Category:      "rent",
			Amount:        10000.0,
			PaymentMethod: "cash",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/expenses", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		ec.CreateExpense(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("CreateExpense Invalid Category", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.CreateExpenseRequest{
			Category:      "luxury_yacht", // invalid
			Amount:        5000.0,
			PaymentMethod: "cash",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/expenses", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		ec.CreateExpense(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("CreateExpense Negative Amount", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Role:   "shop",
		}
		body, _ := json.Marshal(dto.CreateExpenseRequest{
			Category:      "tea_snacks",
			Amount:        -50.0,
			PaymentMethod: "cash",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/expenses", bytes.NewBuffer(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.ContextKeyUser, claims))
		w := httptest.NewRecorder()

		ec.CreateExpense(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ListExpenses Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/expenses", nil)
		w := httptest.NewRecorder()

		ec.ListExpenses(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DeleteExpense Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/shops/me/expenses/exp-123", nil)
		w := httptest.NewRecorder()

		ec.DeleteExpense(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
