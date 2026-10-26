// Package controller handler http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strings"

	"github.com/go-chi/chi/v5"
)

type CategoryController struct {
	service *services.CategoryService
}

func NewCategoryController(service *services.CategoryService) *CategoryController {
	return &CategoryController{service: service}
}

// List handles returning all categories (Public, optional ?shop_id= or ?shop_slug=)
func (c *CategoryController) List(w http.ResponseWriter, r *http.Request) {
	shopID := strings.TrimSpace(r.URL.Query().Get("shop_id"))
	if shopID != "" {
		categories, err := c.service.ListShopCategoriesByShopID(r.Context(), shopID)
		if err == nil {
			reuse.Success(w, "Shop categories retrieved successfully", categories)
			return
		}
	}

	categories, err := c.service.ListCategories(r.Context())
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	reuse.Success(w, "Categories retrieved successfully", categories)
}

// AdminList handles returning all categories with stats (Admin)
func (c *CategoryController) AdminList(w http.ResponseWriter, r *http.Request) {
	categories, err := c.service.AdminListCategories(r.Context())
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	reuse.Success(w, "Admin categories retrieved successfully", categories)
}

// AdminCreate handles creating a category (Admin)
func (c *CategoryController) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cat, err := c.service.CreateCategory(r.Context(), input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Category created successfully", cat)
}

// AdminUpdate handles updating a category (Admin)
func (c *CategoryController) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "category id is required")
		return
	}

	var input dto.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cat, err := c.service.UpdateCategory(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, services.ErrCategorySlugTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Category updated successfully", cat)
}

// AdminDelete handles deleting a category (Admin)
func (c *CategoryController) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "category id is required")
		return
	}

	if err := c.service.DeleteCategory(r.Context(), id); err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Category deleted successfully", nil)
}

// ListMyShopCategories handles returning global + custom categories for the authenticated merchant
func (c *CategoryController) ListMyShopCategories(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	categories, err := c.service.ListShopCategories(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop categories retrieved successfully", categories)
}

// CreateShopCategory handles creating a custom category specifically for the authenticated shop
func (c *CategoryController) CreateShopCategory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cat, err := c.service.CreateShopCategory(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Custom category created successfully", cat)
}

// UpdateShopCategory handles modifying a custom category belonging to the authenticated shop
func (c *CategoryController) UpdateShopCategory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "category id is required")
		return
	}

	var input dto.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cat, err := c.service.UpdateShopCategory(r.Context(), claims.UserID, id, input)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			reuse.Error(w, http.StatusNotFound, "custom category not found or belongs to another shop")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Custom category updated successfully", cat)
}

// DeleteShopCategory handles removing a custom category if no products are assigned to it
func (c *CategoryController) DeleteShopCategory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "category id is required")
		return
	}

	if err := c.service.DeleteShopCategory(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			reuse.Error(w, http.StatusNotFound, "custom category not found or belongs to another shop")
			return
		}
		// If products exist, return conflict 409
		if strings.Contains(err.Error(), "product(s) are assigned") {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Custom category deleted successfully", nil)
}

