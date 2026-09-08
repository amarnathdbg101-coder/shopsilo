// Package controller handles http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

type LoyaltyController struct {
	service *services.LoyaltyService
}

func NewLoyaltyController(service *services.LoyaltyService) *LoyaltyController {
	return &LoyaltyController{
		service: service,
	}
}

// ProcessReturn handles in-store customer returns, restocking inventory (Protected - Shop)
func (c *LoyaltyController) ProcessReturn(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ret, err := c.service.ProcessReturn(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Product return processed successfully. Inventory restocked and points adjusted.", ret)
}

// ListReturns returns the return history for the shopkeeper (Protected - Shop)
func (c *LoyaltyController) ListReturns(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	returns, err := c.service.ListShopReturns(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Returns history retrieved successfully", returns)
}

// CreateOffer allows shopkeeper to create a new in-store offer/discount (Protected - Shop)
func (c *LoyaltyController) CreateOffer(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	offer, err := c.service.CreateOffer(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "In-store offer created successfully", offer)
}

// ListOffers returns all active in-store offers for a shop, indicating VIP unlocks (Public/Customer)
func (c *LoyaltyController) ListOffers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	customerUserID := ""
	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil {
		customerUserID = claims.UserID
	}

	offers, err := c.service.ListShopOffers(r.Context(), slug, customerUserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Store offers retrieved successfully", offers)
}

// ListAllOffers returns all active promotional offers across shops for marketplace deals feed (Public)
func (c *LoyaltyController) ListAllOffers(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	offers, err := c.service.ListAllActiveOffers(r.Context(), category)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Active deals retrieved successfully", offers)
}


// GetUserLoyalty returns the customer's current points and VIP tier status (Protected - Customer)
func (c *LoyaltyController) GetUserLoyalty(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := c.service.GetUserLoyalty(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Loyalty points summary retrieved successfully", summary)
}

// ScanProduct performs instant product lookup via Barcode / SKU / QR (Public)
func (c *LoyaltyController) ScanProduct(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		reuse.Error(w, http.StatusBadRequest, "scanned code is required")
		return
	}

	res, err := c.service.ScanProduct(r.Context(), code)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Product scanned successfully", res)
}
