// Package controller handler http work.
package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

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

	var lat, lng, radiusKm *float64
	if latVal, err := strconv.ParseFloat(query.Get("lat"), 64); err == nil {
		lat = &latVal
	}
	if lngVal, err := strconv.ParseFloat(query.Get("lng"), 64); err == nil {
		lng = &lngVal
	}
	if radVal, err := strconv.ParseFloat(query.Get("radius_km"), 64); err == nil && radVal > 0 {
		radiusKm = &radVal
	} else if radVal, err := strconv.ParseFloat(query.Get("radius"), 64); err == nil && radVal > 0 {
		radiusKm = &radVal
	}
	city := query.Get("city")

	filter := dto.ProductFilter{
		Search:     search,
		CategoryID: categoryID,
		ShopID:     shopID,
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		SortBy:     sortBy,
		Page:       page,
		Limit:      limit,
		Lat:        lat,
		Lng:        lng,
		RadiusKm:   radiusKm,
		City:       city,
	}

	result, err := c.productService.ListProducts(r.Context(), filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Products retrieved successfully", result)
}

// ListMyShopProducts handles retrieving all products belonging strictly to the logged-in shop keeper (Protected - Owner)
func (c *ProductController) ListMyShopProducts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you must register a shop first")
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
		limit = 100
	}

	search := query.Get("q")
	if search == "" {
		search = query.Get("search")
	}

	filter := dto.ProductFilter{
		ShopID:     shop.ID,
		Search:     search,
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
		limit = 16
	}

	search := query.Get("q")
	if search == "" {
		search = query.Get("search")
	}

	filter := dto.ProductFilter{
		ShopID:     shop.ID,
		Search:     search,
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

// FindNearby searches for in-stock products across nearby shops (Public)
func (c *ProductController) FindNearby(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	latStr := query.Get("lat")
	lngStr := query.Get("lng")
	if latStr == "" || lngStr == "" {
		reuse.Error(w, http.StatusBadRequest, "latitude (lat) and longitude (lng) are required query parameters")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid latitude")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid longitude")
		return
	}

	radiusKm, _ := strconv.ParseFloat(query.Get("radius_km"), 64)
	if radiusKm <= 0 {
		radiusKm = 10
	}

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	openNow := query.Get("open_now") == "true" || query.Get("open_now") == "1"

	res, err := c.productService.FindNearbyProducts(
		r.Context(),
		lat, lng, radiusKm,
		query.Get("q"),
		query.Get("category"),
		openNow,
		page, limit,
	)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Nearby in-stock products retrieved successfully", res)
}

// ApplyClearanceMarkdown marks down a slow-moving product and generates a WhatsApp promotional broadcast (Protected - Shop Owner)
func (c *ProductController) ApplyClearanceMarkdown(w http.ResponseWriter, r *http.Request) {
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

	var input dto.ApplyClearanceMarkdownRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.productService.ApplyClearanceMarkdown(r.Context(), claims.UserID, productID, input)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you do not have a registered shop")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Clearance markdown applied and WhatsApp broadcast generated successfully", res)
}

// MakeOffer lets online or mobile shoppers propose a bargained price for a product (Public / Customer).
func (c *ProductController) MakeOffer(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	var input dto.MakeOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.productService.NegotiateBargainOffer(r.Context(), productID, input)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Bargain offer processed successfully", res)
}

// GetPOSBargainAssist provides real-time margin guidance for counter cashiers negotiating in person (Protected - Shop Owner).
func (c *ProductController) GetPOSBargainAssist(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product_id is required")
		return
	}

	proposedPriceStr := r.URL.Query().Get("proposed_price")
	proposedPrice, _ := strconv.ParseFloat(proposedPriceStr, 64)
	if proposedPrice <= 0 {
		reuse.Error(w, http.StatusBadRequest, "proposed_price must be greater than 0")
		return
	}

	res, err := c.productService.GetPOSBargainAssist(r.Context(), claims.UserID, productID, proposedPrice)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you do not have a registered shop")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "POS bargain assist margin advice retrieved successfully", res)
}

// BulkImport handles batch importing 500+ products via CSV file or JSON array (Protected - Shop Owner)
func (c *ProductController) BulkImport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	updateExisting := r.URL.Query().Get("update_existing") == "true" ||
		r.URL.Query().Get("upsert") == "true" ||
		r.FormValue("update_existing") == "true"

	contentType := r.Header.Get("Content-Type")

	// 1. Handle multipart CSV file upload
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(15 * 1024 * 1024); err != nil { // 15MB limit
			reuse.Error(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		var file io.ReadCloser
		for _, key := range []string{"file", "csv", "data", "upload", "spreadsheet"} {
			if f, _, err := r.FormFile(key); err == nil && f != nil {
				file = f
				break
			}
		}

		if file != nil {
			defer file.Close()
			res, err := c.productService.BulkImportProductsFromCSV(r.Context(), claims.UserID, file, updateExisting)
			if err != nil {
				if errors.Is(err, services.ErrShopNotFound) {
					reuse.Error(w, http.StatusNotFound, "you must register a shop first")
					return
				}
				reuse.Error(w, http.StatusBadRequest, err.Error())
				return
			}
			reuse.Created(w, "Bulk product CSV import completed successfully", res)
			return
		}
	}

	// 2. Read Request Body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		reuse.Error(w, http.StatusBadRequest, "request body cannot be empty")
		return
	}

	// 3. Check if body is raw CSV text (Content-Type: text/csv or begins with plain CSV text)
	isRawCSV := strings.Contains(contentType, "text/csv") ||
		strings.Contains(contentType, "text/plain") ||
		(trimmed[0] != '{' && trimmed[0] != '[') ||
		bytes.HasPrefix(trimmed, []byte{0xEF, 0xBB, 0xBF})

	if isRawCSV {
		res, err := c.productService.BulkImportProductsFromCSV(r.Context(), claims.UserID, bytes.NewReader(bodyBytes), updateExisting)
		if err != nil {
			if errors.Is(err, services.ErrShopNotFound) {
				reuse.Error(w, http.StatusNotFound, "you must register a shop first")
				return
			}
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reuse.Created(w, "Bulk product CSV import completed successfully", res)
		return
	}

	// 4. Handle Flexible JSON body (supports numbers as strings, flexible wrappers, etc.)
	items, shouldUpdate, err := parseFlexibleBulkImportJSON(bodyBytes)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if shouldUpdate {
		updateExisting = true
	}

	if len(items) == 0 {
		reuse.Error(w, http.StatusBadRequest, "no valid product rows found in request")
		return
	}

	res, err := c.productService.BulkImportProductsFromJSON(r.Context(), claims.UserID, items, updateExisting)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you must register a shop first")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Bulk product import completed successfully", res)
}

// parseFlexibleBulkImportJSON handles parsing dynamic JSON structures and coerces types cleanly.
func parseFlexibleBulkImportJSON(data []byte) ([]dto.BulkImportProductItem, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var rawData interface{}
	if err := decoder.Decode(&rawData); err != nil {
		return nil, false, err
	}

	var itemsList []interface{}
	var updateExisting bool = false

	switch v := rawData.(type) {
	case []interface{}:
		itemsList = v
	case map[string]interface{}:
		if ue, ok := v["update_existing"].(bool); ok {
			updateExisting = ue
		}
		if items, ok := v["items"].([]interface{}); ok {
			itemsList = items
		} else if prods, ok := v["products"].([]interface{}); ok {
			itemsList = prods
		} else if d, ok := v["data"].([]interface{}); ok {
			itemsList = d
		}
	}

	if len(itemsList) == 0 {
		return nil, false, errors.New("expected array of products or { items: [...] }")
	}

	result := make([]dto.BulkImportProductItem, 0, len(itemsList))
	for _, raw := range itemsList {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		item := dto.BulkImportProductItem{
			Name:          getMapString(m, "name", "title", "product_name", "item_name", "saman"),
			SKU:           getMapString(m, "sku", "code", "item_code", "product_code"),
			Barcode:       getMapString(m, "barcode", "ean", "upc"),
			Price:         getMapFloat(m, "price", "rate", "selling_price", "saleprice"),
			CostPrice:     getMapFloat(m, "cost_price", "cost", "purchase_price", "buy_price", "wholesale_price"),
			ComparePrice:  getMapFloat(m, "compare_price", "mrp", "original_price", "list_price"),
			StockQuantity: getMapInt(m, "stock_quantity", "stock", "quantity", "qty", "inventory"),
			MinStock:      getMapInt(m, "min_stock", "min_quantity", "threshold", "alert_stock"),
			CategoryName:  getMapString(m, "category_name", "category", "cat", "department"),
			CategoryID:    getMapString(m, "category_id"),
			Unit:          getMapString(m, "unit", "uom", "pack"),
			Description:   getMapString(m, "description", "desc", "details"),
		}

		if item.Name != "" || item.Price > 0 {
			result = append(result, item)
		}
	}

	return result, updateExisting, nil
}

func getMapString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if val, exists := m[k]; exists && val != nil {
			str := strings.TrimSpace(fmt.Sprintf("%v", val))
			if str != "" && str != "<nil>" {
				return str
			}
		}
	}
	return ""
}

func getMapFloat(m map[string]interface{}, keys ...string) float64 {
	for _, k := range keys {
		if val, exists := m[k]; exists && val != nil {
			switch v := val.(type) {
			case float64:
				return v
			case json.Number:
				if f, err := v.Float64(); err == nil {
					return f
				}
			case string:
				cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(v, "₹", ""), ",", ""))
				cleaned = strings.TrimPrefix(cleaned, "Rs.")
				cleaned = strings.TrimPrefix(cleaned, "Rs")
				if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func getMapInt(m map[string]interface{}, keys ...string) int {
	for _, k := range keys {
		if val, exists := m[k]; exists && val != nil {
			switch v := val.(type) {
			case int:
				return v
			case float64:
				return int(v)
			case json.Number:
				if i, err := v.Int64(); err == nil {
					return int(i)
				}
			case string:
				cleaned := strings.TrimSpace(strings.ReplaceAll(v, ",", ""))
				if i, err := strconv.Atoi(cleaned); err == nil {
					return i
				}
				if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
					return int(f)
				}
			}
		}
	}
	return 0
}

// DownloadImportTemplate serves a ready-to-use sample CSV import template file (Protected / Public)
func (c *ProductController) DownloadImportTemplate(w http.ResponseWriter, r *http.Request) {
	csvBytes := c.productService.GenerateCSVImportTemplate()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="shopsilo_products_import_template.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBytes)
}


