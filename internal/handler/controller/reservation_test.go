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

func TestReservationControllerValidation(t *testing.T) {
	rc := controller.NewReservationController(nil)

	t.Run("Create Reservation Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReservationRequest{
			ProductID: "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
			Quantity:  1,
		})
		req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		rc.Create(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Create Reservation Invalid Product ID", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreateReservationRequest{
			ProductID: "invalid-uuid",
			Quantity:  1,
		})
		req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Create Reservation Invalid Quantity (0 or negative)", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.CreateReservationRequest{
			ProductID: "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
			Quantity:  0,
		})
		req := httptest.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ListUserReservations Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/reservations", nil)
		w := httptest.NewRecorder()

		rc.ListUserReservations(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetByID Missing ID", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		req := httptest.NewRequest(http.MethodGet, "/reservations/", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.GetByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("CancelUserReservation Missing ID", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "customer@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		req := httptest.NewRequest(http.MethodPost, "/reservations//cancel", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.CancelUserReservation(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("VerifyShopReservation Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.VerifyReservationRequest{
			PickupCode: "123456",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/reservations/verify", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		rc.VerifyShopReservation(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("VerifyShopReservation Missing Code and Number", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "shop-owner-123",
			Email:  "owner@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.VerifyReservationRequest{
			PickupCode:        "",
			ReservationNumber: "",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/reservations/verify", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		rc.VerifyShopReservation(w, req)

		// Handler checks if both are empty and returns 400
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}
