package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"testing"
)

// TestUploadControllerDeepFeatures tests image upload boundaries & banner constraints
func TestUploadControllerDeepFeatures(t *testing.T) {
	upc := controller.NewUploadController(nil)

	claims := &middleware.Claims{
		UserID: "merchant-uid-101",
		Email:  "merchant@shopsilo.in",
		Role:   "merchant",
	}
	ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

	t.Run("Shop Images - Reject more than 4 banners", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add 5 banners
		for i := 1; i <= 5; i++ {
			part, err := writer.CreateFormFile("banners", "banner.jpg")
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			_, _ = part.Write([]byte("fake-banner-content"))
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/shops/me/images", body).WithContext(ctx)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		upc.UploadShopImages(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for 5 banners, got %d (body: %s)", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Fatalf("expected success to be false")
		}
	})

	t.Run("Shop Images - Reject when neither logo nor banners provided", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/shops/me/images", body).WithContext(ctx)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		upc.UploadShopImages(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request when empty, got %d", w.Code)
		}
	})

	t.Run("Product Images - Reject empty form without images", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/products/images", body).WithContext(ctx)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		upc.UploadProductImages(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request when no product images provided, got %d", w.Code)
		}
	})

	t.Run("ServeImage - 404 for missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/images/", nil)
		w := httptest.NewRecorder()

		upc.ServeImage(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found for missing key, got %d", w.Code)
		}
	})
}

// TestShopBannersLimitValidation verifies DTO validator enforces up to 4 banners
func TestShopBannersLimitValidation(t *testing.T) {
	t.Run("Create Shop with 1 to 4 Banners is valid", func(t *testing.T) {
		req1 := &dto.CreateShopRequest{
			Name:     "Kalyan Textiles",
			Category: "Clothing",
			Address:  "Gandhi Chowk, Darbhanga",
			Banners: []string{
				"https://example.com/b1.jpg",
			},
		}
		if err := reuse.ValidateStruct(req1); err != nil {
			t.Fatalf("expected 1 banner to be valid, got: %v", err)
		}

		req4 := &dto.CreateShopRequest{
			Name:     "Kalyan Textiles",
			Category: "Clothing",
			Address:  "Gandhi Chowk, Darbhanga",
			Banners: []string{
				"https://example.com/b1.jpg",
				"https://example.com/b2.jpg",
				"https://example.com/b3.jpg",
				"https://example.com/b4.jpg",
			},
		}
		if err := reuse.ValidateStruct(req4); err != nil {
			t.Fatalf("expected 4 banners to be valid, got: %v", err)
		}
	})

	t.Run("Create Shop with 5 Banners is rejected by validator", func(t *testing.T) {
		req5 := &dto.CreateShopRequest{
			Name:     "Kalyan Textiles",
			Category: "Clothing",
			Address:  "Gandhi Chowk, Darbhanga",
			Banners: []string{
				"https://example.com/b1.jpg",
				"https://example.com/b2.jpg",
				"https://example.com/b3.jpg",
				"https://example.com/b4.jpg",
				"https://example.com/b5.jpg",
			},
		}
		if err := reuse.ValidateStruct(req5); err == nil {
			t.Fatalf("expected 5 banners to fail validation, but it passed")
		}
	})

	t.Run("Update Shop with 5 Banners is rejected by validator", func(t *testing.T) {
		banners := []string{
			"https://example.com/1.jpg",
			"https://example.com/2.jpg",
			"https://example.com/3.jpg",
			"https://example.com/4.jpg",
			"https://example.com/5.jpg",
		}
		reqUpdate := &dto.UpdateShopRequest{
			Banners: &banners,
		}
		if err := reuse.ValidateStruct(reqUpdate); err == nil {
			t.Fatalf("expected 5 banners in update to fail validation, but it passed")
		}
	})

	t.Run("Update Shop with 4 Banners is valid", func(t *testing.T) {
		banners := []string{
			"https://example.com/1.jpg",
			"https://example.com/2.jpg",
			"https://example.com/3.jpg",
			"https://example.com/4.jpg",
		}
		reqUpdate := &dto.UpdateShopRequest{
			Banners: &banners,
		}
		if err := reuse.ValidateStruct(reqUpdate); err != nil {
			t.Fatalf("expected 4 banners in update to be valid, got: %v", err)
		}
	})
}
