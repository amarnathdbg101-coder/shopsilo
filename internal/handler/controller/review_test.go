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

func TestReviewControllerValidation(t *testing.T) {
	rc := controller.NewReviewController(nil)

	t.Run("AddOrUpdate Review Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReviewRequest{
			Rating:  5,
			Comment: "Great store!",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/delhi-store/reviews", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		rc.AddOrUpdate(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("AddOrUpdate Review Invalid Rating (outside 1-5)", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreateReviewRequest{
			Rating:  6, // Maximum is 5
			Comment: "Invalid rating",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/delhi-store/reviews", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.AddOrUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for rating > 5, got %d", w.Code)
		}
	})

	t.Run("AddOrUpdate Review Zero Rating", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreateReviewRequest{
			Rating:  0, // Minimum is 1
			Comment: "Invalid rating",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/delhi-store/reviews", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.AddOrUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for rating 0, got %d", w.Code)
		}
	})

	t.Run("List Reviews Missing Slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops//reviews", nil)
		w := httptest.NewRecorder()

		rc.List(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing slug, got %d", w.Code)
		}
	})

	t.Run("Delete Review Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/shops/delhi-store/reviews", nil)
		w := httptest.NewRecorder()

		rc.Delete(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
