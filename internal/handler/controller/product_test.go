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

func TestProductControllerValidation(t *testing.T) {
	pc := controller.NewProductController(nil, nil)

	t.Run("Create Product Unauthorized if no claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateProductRequest{
			Name:          "Wireless Mouse",
			SKU:           "WM-001",
			Price:         29.99,
			CategoryID:    "cat-1",
			StockQuantity: 10,
		})
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.Create(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Create Product Invalid Input", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Invalid: Name too short, price is zero, category is empty
		invalidBody, _ := json.Marshal(dto.CreateProductRequest{
			Name:       "A",
			SKU:        "",
			Price:      0,
			CategoryID: "",
		})

		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(invalidBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Fatalf("expected success to be false")
		}
	})

	t.Run("Create Product Max 4 Images Exceeded", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// 5 images exceeds the professional limit of 4
		fiveImagesBody, _ := json.Marshal(dto.CreateProductRequest{
			Name:          "Wireless Mouse",
			SKU:           "WM-001",
			Price:         29.99,
			CategoryID:    "cat-1",
			StockQuantity: 10,
			Images: []string{
				"https://example.com/1.jpg",
				"https://example.com/2.jpg",
				"https://example.com/3.jpg",
				"https://example.com/4.jpg",
				"https://example.com/5.jpg",
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(fiveImagesBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.Create(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request when exceeding 4 images, got %d", w.Code)
		}
	})

	t.Run("GetByID Missing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/", nil)
		w := httptest.NewRecorder()

		pc.GetByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("GetBySlug Missing Slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/slug/", nil)
		w := httptest.NewRecorder()

		pc.GetBySlug(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}
