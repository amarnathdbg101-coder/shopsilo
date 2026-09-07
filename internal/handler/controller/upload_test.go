package controller_test

import (
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"testing"
)

func TestUploadControllerValidation(t *testing.T) {
	upc := controller.NewUploadController(nil)

	t.Run("Upload Avatar Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/user/avatar", nil)
		w := httptest.NewRecorder()

		upc.UploadUserAvatar(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Upload Shop Images Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/shops/me/images", nil)
		w := httptest.NewRecorder()

		upc.UploadShopImages(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Upload Product Images Unauthorized if no claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products/images", nil)
		w := httptest.NewRecorder()

		upc.UploadProductImages(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}
