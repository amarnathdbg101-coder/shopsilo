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

	t.Run("FindNearby Missing Coords", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/products/nearby", nil)
		w := httptest.NewRecorder()

		pc.FindNearby(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing coordinates, got %d", w.Code)
		}
	})

	t.Run("ApplyClearanceMarkdown Unauthorized if no claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.ApplyClearanceMarkdownRequest{
			DiscountPercentage: 25.0,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/products/prod-123/markdown", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.ApplyClearanceMarkdown(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ApplyClearanceMarkdown Invalid Discount Percentage", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		body, _ := json.Marshal(dto.ApplyClearanceMarkdownRequest{
			DiscountPercentage: 150.0, // Invalid: must be < 100
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/products/prod-123/markdown", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.ApplyClearanceMarkdown(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for discount > 100, got %d", w.Code)
		}
	})

	t.Run("MakeOffer Invalid Phone and Non-Positive Offer", func(t *testing.T) {
		body, _ := json.Marshal(dto.MakeOfferRequest{
			OfferedPrice:  0, // Invalid: must be gt 0
			CustomerPhone: "123", // Invalid: min 10
		})
		req := httptest.NewRequest(http.MethodPost, "/products/prod-1/make-offer", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		pc.MakeOffer(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid offer, got %d", w.Code)
		}
	})

	t.Run("GetPOSBargainAssist Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/bargain-assist?product_id=prod-1&proposed_price=150", nil)
		w := httptest.NewRecorder()

		pc.GetPOSBargainAssist(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetPOSBargainAssist Missing Query Params", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user-123",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		// Missing product_id
		req := httptest.NewRequest(http.MethodGet, "/shops/me/pos/bargain-assist?proposed_price=150", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		pc.GetPOSBargainAssist(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing product_id, got %d", w.Code)
		}
	})
}

