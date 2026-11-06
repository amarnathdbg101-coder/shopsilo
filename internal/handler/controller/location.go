// Package controller handles HTTP request parsing and response delivery.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

// LocationController handles endpoints for multi-store godowns, variant matrices, and stock transfers.
type LocationController struct {
	service *services.LocationService
}

// NewLocationController initializes the location controller.
func NewLocationController(service *services.LocationService) *LocationController {
	return &LocationController{
		service: service,
	}
}

// ListLocations returns all showroom and godown facilities of the shop.
func (c *LocationController) ListLocations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	locations, err := c.service.ListLocations(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Locations retrieved successfully", locations)
}

// CreateLocation creates a new godown or showroom facility.
func (c *LocationController) CreateLocation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	location, err := c.service.CreateLocation(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Location created successfully", location)
}

// UpdateLocation modifies details of an existing location.
func (c *LocationController) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	locationID := chi.URLParam(r, "id")
	if locationID == "" {
		reuse.Error(w, http.StatusBadRequest, "missing location id")
		return
	}

	var input dto.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	location, err := c.service.UpdateLocation(r.Context(), claims.UserID, locationID, input)
	if err != nil {
		if errors.Is(err, services.ErrLocationNotFound) {
			reuse.Error(w, http.StatusNotFound, "location not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Location updated successfully", location)
}

// GetLocationInventory lists physical stock counts for a specific godown/showroom.
func (c *LocationController) GetLocationInventory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	locationID := chi.URLParam(r, "id")
	if locationID == "" {
		reuse.Error(w, http.StatusBadRequest, "missing location id")
		return
	}

	items, err := c.service.GetLocationInventory(r.Context(), claims.UserID, locationID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Location inventory retrieved", items)
}

// AdjustStock adjusts stock counts with row-level locking.
func (c *LocationController) AdjustStock(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.AdjustLocationStockRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.AdjustStock(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Stock adjusted successfully", nil)
}

// ListVariants returns all size/color/shade SKUs of a parent product.
func (c *LocationController) ListVariants(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "productId")
	locationID := r.URL.Query().Get("location_id")

	variants, err := c.service.ListVariants(r.Context(), claims.UserID, productID, locationID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product variants retrieved", variants)
}

// GetPublicVariants returns all active variants for a product on the public customer storefront.
func (c *LocationController) GetPublicVariants(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product_id is required")
		return
	}

	variants, err := c.service.ListVariantsPublic(r.Context(), productID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product variants retrieved", variants)
}

// CreateVariant creates an individual variant SKU.
func (c *LocationController) CreateVariant(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "productId")
	var input dto.CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	variant, err := c.service.CreateVariant(r.Context(), claims.UserID, productID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Product variant created", variant)
}

// GenerateVariantMatrix auto-creates all Size x Color or Shade combinations in 1 tap.
func (c *LocationController) GenerateVariantMatrix(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "productId")
	var input dto.GenerateVariantMatrixRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	variants, err := c.service.GenerateVariantMatrix(r.Context(), claims.UserID, productID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Variant matrix generated successfully", variants)
}

// UpdateVariant updates a variant SKU.
func (c *LocationController) UpdateVariant(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID := chi.URLParam(r, "variantId")
	var input dto.UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	variant, err := c.service.UpdateVariant(r.Context(), claims.UserID, variantID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Variant updated successfully", variant)
}

// DeleteVariant deactivates a variant SKU.
func (c *LocationController) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID := chi.URLParam(r, "variantId")
	err := c.service.DeleteVariant(r.Context(), claims.UserID, variantID)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Variant deleted successfully", nil)
}

// CreateTransfer creates a stock transfer requisition between godown and showroom.
func (c *LocationController) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateStockTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	transfer, err := c.service.CreateStockTransfer(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Stock transfer requisition created", transfer)
}

// ListTransfers lists stock transfers with optional status filter.
func (c *LocationController) ListTransfers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var statusFilter *model.TransferStatus
	statusParam := r.URL.Query().Get("status")
	if statusParam != "" {
		st := model.TransferStatus(statusParam)
		statusFilter = &st
	}

	transfers, err := c.service.ListTransfers(r.Context(), claims.UserID, statusFilter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Stock transfers retrieved", transfers)
}

// GetTransfer fetches details of a single stock transfer.
func (c *LocationController) GetTransfer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	transferID := chi.URLParam(r, "id")
	transfer, err := c.service.GetTransfer(r.Context(), claims.UserID, transferID)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, "transfer not found")
		return
	}

	reuse.Success(w, "Stock transfer details", transfer)
}

// UpdateTransferStatus advances a transfer (DISPATCHED, IN_TRANSIT, RECEIVED, RECONCILED).
func (c *LocationController) UpdateTransferStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	transferID := chi.URLParam(r, "id")
	var input dto.UpdateTransferStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	transfer, err := c.service.UpdateTransferStatus(r.Context(), claims.UserID, transferID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Stock transfer status updated", transfer)
}
