// Package controller handler http work.
package controller

import (
	"net/http"
	"shopMe/internal/handler/services"
	"shopMe/internal/reuse"
)

type CategoryController struct {
	service *services.CategoryService
}

func NewCategoryController(service *services.CategoryService) *CategoryController {
	return &CategoryController{service: service}
}

// List handles returning all categories (Public)
func (c *CategoryController) List(w http.ResponseWriter, r *http.Request) {
	categories, err := c.service.ListCategories(r.Context())
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	reuse.Success(w, "Categories retrieved successfully", categories)
}
