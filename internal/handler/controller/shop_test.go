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

func TestShopControllerValidation(t *testing.T) {
	sc := controller.NewShopController(nil)

	t.Run("Create Shop Unauthorized if no claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateShopRequest{
			Name:     "Electronics Hub",
			Category: "Electronics",
			Address:  "Main Road, City",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		sc.Create(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Create Shop Invalid Input", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		invalidBody, _ := json.Marshal(dto.CreateShopRequest{
			Name:     "A", // min 2
			Category: "",  // required
			Address:  "12", // min 5
		})

		req := httptest.NewRequest(http.MethodPost, "/shops", bytes.NewBuffer(invalidBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		sc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Fatalf("expected success to be false")
		}
	})

	t.Run("GetMyShop Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me", nil)
		w := httptest.NewRecorder()

		sc.GetMyShop(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("UpdateMyShop Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/shops/me", bytes.NewBufferString("{}"))
		w := httptest.NewRecorder()

		sc.UpdateMyShop(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DeleteMyShop Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/shops/me", nil)
		w := httptest.NewRecorder()

		sc.DeleteMyShop(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetByID Missing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/", nil)
		w := httptest.NewRecorder()

		sc.GetByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ToggleStatus Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/shops/me/status", bytes.NewBufferString(`{"is_open":false}`))
		w := httptest.NewRecorder()

		sc.ToggleStatus(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetBySlug Missing Slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/slug/", nil)
		w := httptest.NewRecorder()

		sc.GetBySlug(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetShopQR Missing Slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops//qr", nil)
		w := httptest.NewRecorder()

		sc.GetShopQR(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetMyShopQR Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/qr", nil)
		w := httptest.NewRecorder()

		sc.GetMyShopQR(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Create Shop Invalid Coordinates", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "customer",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		invalidLat := 95.0 // Latitude must be between -90 and 90
		invalidBody, _ := json.Marshal(dto.CreateShopRequest{
			Name:     "Valid Shop Name",
			Category: "Retail",
			Address:  "123 Market Street",
			Latitude: &invalidLat,
		})

		req := httptest.NewRequest(http.MethodPost, "/shops", bytes.NewBuffer(invalidBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		sc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid latitude, got %d", w.Code)
		}
	})

	t.Run("GetDailyDigest Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/digest", nil)
		w := httptest.NewRecorder()

		sc.GetDailyDigest(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}

