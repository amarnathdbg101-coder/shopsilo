package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"testing"
)

func TestCategoryControllerValidation(t *testing.T) {
	catc := controller.NewCategoryController(nil)

	t.Run("AdminCreate Category Validation Failure (Empty Name)", func(t *testing.T) {
		invalidBody, _ := json.Marshal(dto.CreateCategoryRequest{
			Name: "", // required
		})

		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(invalidBody))
		w := httptest.NewRecorder()

		catc.AdminCreate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("AdminUpdate Category Validation Failure (Missing ID)", func(t *testing.T) {
		validBody, _ := json.Marshal(dto.UpdateCategoryRequest{
			Name: "Updated Grocery",
		})

		req := httptest.NewRequest(http.MethodPut, "/admin/categories/", bytes.NewBuffer(validBody))
		w := httptest.NewRecorder()

		catc.AdminUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("AdminDelete Category Validation Failure (Missing ID)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/categories/", nil)
		w := httptest.NewRecorder()

		catc.AdminDelete(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ListMyShopCategories Unauthorized when claims missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/merchant/categories", nil)
		w := httptest.NewRecorder()

		catc.ListMyShopCategories(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("CreateShopCategory Unauthorized when claims missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/merchant/categories", bytes.NewBuffer([]byte(`{"name":"Snacks"}`)))
		w := httptest.NewRecorder()

		catc.CreateShopCategory(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("UpdateShopCategory Unauthorized when claims missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/merchant/categories/123", bytes.NewBuffer([]byte(`{"name":"Snacks"}`)))
		w := httptest.NewRecorder()

		catc.UpdateShopCategory(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DeleteShopCategory Unauthorized when claims missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/merchant/categories/123", nil)
		w := httptest.NewRecorder()

		catc.DeleteShopCategory(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})
}
