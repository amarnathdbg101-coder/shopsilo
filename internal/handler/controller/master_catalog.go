package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
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
	claims, ok := middleware.RequireUserClaims(w, r)
	if !ok {
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
	if _, ok := middleware.RequireUserClaims(w, r); !ok {
		return
	}

	req, ok := reuse.DecodeJSON[dto.MasterCatalogBulkSeedRequest](w, r)
	if !ok {
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
	if _, ok := middleware.RequireUserClaims(w, r); !ok {
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

// ListMasterProducts handles GET /admin/catalog/products
func (c *MasterCatalogController) ListMasterProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	brand := q.Get("brand")
	category := q.Get("category")

	limit := 50
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(q.Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	var isVerified *bool
	if v := q.Get("is_verified"); v != "" {
		parsed := v == "true" || v == "1"
		isVerified = &parsed
	}

	products, total, err := c.catalogService.ListMasterProducts(r.Context(), search, brand, category, isVerified, limit, offset)
	if err != nil {
		c.logger.Error("list master products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to list master products: "+err.Error())
		return
	}

	reuse.Success(w, "Master products retrieved", map[string]interface{}{
		"products": products,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateMasterProduct handles POST /admin/catalog/products
func (c *MasterCatalogController) CreateMasterProduct(w http.ResponseWriter, r *http.Request) {
	p, ok := reuse.DecodeJSON[model.MasterCatalogProduct](w, r)
	if !ok {
		return
	}

	created, err := c.catalogService.CreateMasterProduct(r.Context(), &p)
	if err != nil {
		c.logger.Error("create master product failed", zap.Error(err))
		reuse.Error(w, http.StatusBadRequest, "failed to create master product: "+err.Error())
		return
	}

	reuse.Success(w, "Master product created successfully", created)
}

// UpdateMasterProduct handles PUT /admin/catalog/products/{id}
func (c *MasterCatalogController) UpdateMasterProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	p, ok := reuse.DecodeJSON[model.MasterCatalogProduct](w, r)
	if !ok {
		return
	}

	updated, err := c.catalogService.UpdateMasterProduct(r.Context(), id, &p)
	if err != nil {
		c.logger.Error("update master product failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to update master product: "+err.Error())
		return
	}

	reuse.Success(w, "Master product updated successfully", updated)
}

// DeleteMasterProduct handles DELETE /admin/catalog/products/{id}
func (c *MasterCatalogController) DeleteMasterProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := c.catalogService.DeleteMasterProduct(r.Context(), id); err != nil {
		c.logger.Error("delete master product failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to delete master product: "+err.Error())
		return
	}

	reuse.Success(w, "Master product deleted successfully", map[string]string{"deleted_id": id})
}

// GetMasterStats handles GET /admin/catalog/stats
func (c *MasterCatalogController) GetMasterStats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.catalogService.GetMasterStats(r.Context())
	if err != nil {
		c.logger.Error("get master stats failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to get stats: "+err.Error())
		return
	}
	reuse.Success(w, "Master catalog stats retrieved", stats)
}

// ListSoftDeletedProducts handles GET /admin/products/soft-deleted
func (c *MasterCatalogController) ListSoftDeletedProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	limit := 50
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(q.Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	items, total, err := c.catalogService.ListSoftDeletedProducts(r.Context(), search, limit, offset)
	if err != nil {
		c.logger.Error("list soft deleted products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to list soft deleted products: "+err.Error())
		return
	}

	reuse.Success(w, "Soft-deleted products retrieved", map[string]interface{}{
		"products": items,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// RestoreProduct handles POST /admin/products/{id}/restore
func (c *MasterCatalogController) RestoreProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := c.catalogService.RestoreProduct(r.Context(), id); err != nil {
		c.logger.Error("restore product failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to restore product: "+err.Error())
		return
	}

	reuse.Success(w, "Product restored to active store catalog", map[string]string{"restored_id": id})
}

type batchIDsRequest struct {
	IDs []string `json:"ids"`
}

// AdminBatchDeleteMasterProducts handles POST /admin/catalog/products/batch
func (c *MasterCatalogController) AdminBatchDeleteMasterProducts(w http.ResponseWriter, r *http.Request) {
	req, ok := reuse.DecodeJSON[batchIDsRequest](w, r)
	if !ok {
		return
	}
	if len(req.IDs) == 0 {
		reuse.Error(w, http.StatusBadRequest, "ids array is required")
		return
	}
	count, err := c.catalogService.BatchDeleteMasterProducts(r.Context(), req.IDs)
	if err != nil {
		c.logger.Error("batch delete master products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "batch delete failed: "+err.Error())
		return
	}
	reuse.Success(w, "Selected master products deleted", map[string]int64{"deleted_count": count})
}

// AdminDeleteAllMasterProducts handles DELETE /admin/catalog/products/all
func (c *MasterCatalogController) AdminDeleteAllMasterProducts(w http.ResponseWriter, r *http.Request) {
	count, err := c.catalogService.DeleteAllMasterProducts(r.Context())
	if err != nil {
		c.logger.Error("delete all master products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "delete all failed: "+err.Error())
		return
	}
	reuse.Success(w, "All master products deleted", map[string]int64{"deleted_count": count})
}

// AdminBatchHardDeleteProducts handles POST /admin/products/batch
func (c *MasterCatalogController) AdminBatchHardDeleteProducts(w http.ResponseWriter, r *http.Request) {
	req, ok := reuse.DecodeJSON[batchIDsRequest](w, r)
	if !ok {
		return
	}
	if len(req.IDs) == 0 {
		reuse.Error(w, http.StatusBadRequest, "ids array is required")
		return
	}
	count, err := c.catalogService.BatchHardDeleteProducts(r.Context(), req.IDs)
	if err != nil {
		c.logger.Error("batch hard delete products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "batch hard delete failed: "+err.Error())
		return
	}
	reuse.Success(w, "Selected products permanently deleted", map[string]int64{"deleted_count": count})
}

// AdminHardDeleteAllSoftDeletedProducts handles DELETE /admin/products/all
func (c *MasterCatalogController) AdminHardDeleteAllSoftDeletedProducts(w http.ResponseWriter, r *http.Request) {
	count, err := c.catalogService.HardDeleteAllSoftDeletedProducts(r.Context())
	if err != nil {
		c.logger.Error("hard delete all soft-deleted products failed", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "delete all failed: "+err.Error())
		return
	}
	reuse.Success(w, "All soft-deleted products permanently deleted", map[string]int64{"deleted_count": count})
}
