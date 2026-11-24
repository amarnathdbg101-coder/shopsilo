package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"shopMe/internal/handler/services"
)

// MasterCatalogController handles HTTP requests for master catalog lookup and admin seeding.
type MasterCatalogController struct {
	catalogService *services.MasterCatalogService
	logger         *zap.Logger
}

func NewMasterCatalogController(
	catalogService *services.MasterCatalogService,
	logger *zap.Logger,
) *MasterCatalogController {
	return &MasterCatalogController{
		catalogService: catalogService,
		logger:         logger,
	}
}

// Lookup handles GET /shops/me/catalog/lookup/{barcode} (Protected - Merchant)
func (c *MasterCatalogController) Lookup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	barcode := chi.URLParam(r, "barcode")
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		reuse.Error(w, http.StatusBadRequest, "barcode parameter is required")
		return
	}

	result, err := c.catalogService.LookupByBarcode(r.Context(), claims.UserID, barcode)
	if err != nil {
		c.logger.Error("master catalog lookup error", zap.Error(err), zap.String("barcode", barcode))
		reuse.Error(w, http.StatusInternalServerError, "failed to lookup catalog: "+err.Error())
		return
	}

	reuse.Success(w, "Catalog lookup completed", result)
}

// BulkSeed handles POST /admin/catalog/bulk-seed (Protected - Admin Only)
func (c *MasterCatalogController) BulkSeed(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.MasterCatalogBulkSeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	result, err := c.catalogService.BulkSeed(r.Context(), &req)
	if err != nil {
		c.logger.Error("master catalog bulk seed error", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "bulk seed failed: "+err.Error())
		return
	}

	reuse.Success(w, "Master catalog seeded successfully", result)
}

// AdminHardDeleteProduct handles DELETE /admin/products/{id} (Protected - Admin Only)
func (c *MasterCatalogController) AdminHardDeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	if strings.TrimSpace(productID) == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := c.catalogService.AdminHardDeleteProduct(r.Context(), productID); err != nil {
		c.logger.Error("admin hard delete product failed", zap.Error(err), zap.String("product_id", productID))
		reuse.Error(w, http.StatusInternalServerError, "hard delete failed: "+err.Error())
		return
	}

	reuse.Success(w, "Product permanently deleted by admin", map[string]string{
		"deleted_id": productID,
	})
}
