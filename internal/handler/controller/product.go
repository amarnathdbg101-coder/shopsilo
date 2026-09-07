// Package controller handler http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ProductController struct {
	productService *services.ProductService
	shopService    *services.ShopService
}

func NewProductController(
	productService *services.ProductService,
	shopService *services.ShopService,
) *ProductController {
	return &ProductController{
		productService: productService,
		shopService:    shopService,
	}
}

// Create handles creating a new product (Protected - Shop Owner)
func (c *ProductController) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := c.productService.CreateProduct(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrTooManyProductImages) || errors.Is(err, services.ErrInvalidCategory) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, repository.ErrSKUTaken) || errors.Is(err, repository.ErrProductSlugTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Product created successfully", product)
}

// Update handles updating an existing product (Protected - Shop Owner)
func (c *ProductController) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	var input dto.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := c.productService.UpdateProduct(r.Context(), claims.UserID, productID, input)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, services.ErrTooManyProductImages) || errors.Is(err, services.ErrInvalidCategory) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, repository.ErrSKUTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product updated successfully", updated)
}

// Delete handles deleting a product (Protected - Shop Owner)
func (c *ProductController) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := c.productService.DeleteProduct(r.Context(), claims.UserID, productID); err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product deleted successfully", nil)
}

// GetByID handles retrieving a single product by UUID (Public)
func (c *ProductController) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	product, err := c.productService.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product retrieved successfully", product)
}

// GetBySlug handles retrieving a single product by SEO slug (Public)
func (c *ProductController) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "product slug is required")
		return
	}

	product, err := c.productService.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product retrieved successfully", product)
}

// List handles searching, filtering, and paginating products (Public)
func (c *ProductController) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := query.Get("q")
	if search == "" {
		search = query.Get("search")
	}
	categoryID := query.Get("category_id")
	shopID := query.Get("shop_id")
	sortBy := query.Get("sort_by")

	minPrice, _ := strconv.ParseFloat(query.Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(query.Get("max_price"), 64)

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 12
	}

	filter := dto.ProductFilter{
		Search:     search,
		CategoryID: categoryID,
		ShopID:     shopID,
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		SortBy:     sortBy,
		Page:       page,
		Limit:      limit,
	}

	result, err := c.productService.ListProducts(r.Context(), filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Products retrieved successfully", result)
}

// ListByShop handles retrieving all products for a specific shop by slug (Public)
func (c *ProductController) ListByShop(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	shop, err := c.shopService.GetShopBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 12
	}

	filter := dto.ProductFilter{
		ShopID:     shop.ID,
		Search:     query.Get("q"),
		CategoryID: query.Get("category_id"),
		SortBy:     query.Get("sort_by"),
		Page:       page,
		Limit:      limit,
	}

	result, err := c.productService.ListProducts(r.Context(), filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop products retrieved successfully", result)
}
