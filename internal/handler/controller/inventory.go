// Package controller handles http work.
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type InventoryController struct {
	service *services.InventoryService
}

func NewInventoryController(service *services.InventoryService) *InventoryController {
	return &InventoryController{
		service: service,
	}
}

// AdjustStock updates stock on wholesale arrival (+qty) or manual adjustments (-qty) (Protected - Shop Owner)
func (c *InventoryController) AdjustStock(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	inv, err := c.service.AdjustStock(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Inventory stock updated successfully", inv)
}

// GetLowStockAlerts lists all products running low on stock for the shopkeeper (Protected - Shop Owner)
func (c *InventoryController) GetLowStockAlerts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetLowStockAlerts(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Low stock alerts retrieved successfully", res)
}

// DownloadReorderSheetPDF generates and downloads the ready-to-print Wholesale Re-order PDF sheet (Protected - Shop Owner)
func (c *InventoryController) DownloadReorderSheetPDF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	pdfBytes, err := c.service.GenerateReorderSheetPDF(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("reorder-sheet-%s.pdf", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// GetSupplierReorderWhatsApp generates a WhatsApp Click-to-Chat link to send a wholesale reorder to a supplier (Protected - Shop Owner)
func (c *InventoryController) GetSupplierReorderWhatsApp(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	supplierPhone := r.URL.Query().Get("supplier_phone")
	res, err := c.service.GenerateSupplierReorderWhatsApp(r.Context(), claims.UserID, supplierPhone)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Supplier WhatsApp reorder link generated", res)
}

// SubscribeStockAlert registers customer interest for out-of-stock items (Public / Shopper)
func (c *InventoryController) SubscribeStockAlert(w http.ResponseWriter, r *http.Request) {
	productID := strings.TrimSpace(chi.URLParam(r, "id"))
	if productID == "" {
		reuse.Error(w, http.StatusBadRequest, "product id is required")
		return
	}

	var input dto.CreateStockAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.SubscribeStockAlert(r.Context(), productID, input)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			reuse.Error(w, http.StatusNotFound, "product not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Stock alert subscription confirmed. You will be notified when this item is back in stock!", map[string]string{
		"product_id":     productID,
		"customer_phone": input.CustomerPhone,
	})
}

// GetDemandWatchlist returns the list of out-of-stock items with unmet customer demand (Protected - Shop Owner)
func (c *InventoryController) GetDemandWatchlist(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetDemandWatchlist(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Customer demand watchlist retrieved successfully", res)
}

