// Package controller handles HTTP requests for Scan & Go and customer self-checkout.
package controller

import (
	"encoding/json"
	"net/http"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

// ScanAndGoController handles self-checkout cart endpoints.
type ScanAndGoController struct {
	service *services.ScanAndGoService
}

// NewScanAndGoController initializes ScanAndGoController.
func NewScanAndGoController(service *services.ScanAndGoService) *ScanAndGoController {
	return &ScanAndGoController{
		service: service,
	}
}

// StartSession creates a new Scan & Go cart session for a customer in the shop.
func (c *ScanAndGoController) StartSession(w http.ResponseWriter, r *http.Request) {
	var input dto.StartScanSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var customerUserID *string
	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil && claims.UserID != "" {
		customerUserID = &claims.UserID
	}

	res, err := c.service.StartSession(r.Context(), customerUserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Scan & Go cart session started", res)
}

// AddItem adds a scanned item into the customer's cart.
func (c *ScanAndGoController) AddItem(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		reuse.Error(w, http.StatusBadRequest, "session id is required")
		return
	}

	var input dto.AddItemToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.AddItemToCart(r.Context(), sessionID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Item added to cart", res)
}

// UpdateItem updates quantity or removes an item.
func (c *ScanAndGoController) UpdateItem(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	if sessionID == "" || itemID == "" {
		reuse.Error(w, http.StatusBadRequest, "session id and item id are required")
		return
	}

	var input dto.UpdateItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := c.service.UpdateItemQuantity(r.Context(), sessionID, itemID, input.Quantity)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Cart item updated", res)
}

// GetSessionDetails returns the customer's active cart.
func (c *ScanAndGoController) GetSessionDetails(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		reuse.Error(w, http.StatusBadRequest, "session id is required")
		return
	}

	res, err := c.service.GetSessionDetails(r.Context(), sessionID)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Cart retrieved", res)
}

// VerifyAtCounter allows cashier to peek customer cart by 6-digit code or QR token.
func (c *ScanAndGoController) VerifyAtCounter(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		reuse.Error(w, http.StatusBadRequest, "session code or token is required")
		return
	}

	res, err := c.service.VerifySessionAtCounter(r.Context(), claims.UserID, code)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Customer cart verified at counter", res)
}

// ConvertToBill converts the customer's cart into an official POS bill with 1-tap checkout.
func (c *ScanAndGoController) ConvertToBill(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		reuse.Error(w, http.StatusBadRequest, "session code or token is required")
		return
	}

	var input dto.ConvertToBillRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Default to cash if empty body
		input.PaymentMethod = "cash"
	}

	res, err := c.service.ConvertToBill(r.Context(), claims.UserID, code, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Scan & Go cart successfully converted into POS bill", res)
}
