// Package controller handles HTTP requests for margin shield and counter speed hotkeys.
package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

// MarginController handles counter pricing guards and rapid hotkeys.
type MarginController struct {
	service *services.MarginService
}

// NewMarginController creates a new MarginController instance.
func NewMarginController(service *services.MarginService) *MarginController {
	return &MarginController{
		service: service,
	}
}

// CheckMargin validates bargaining proposed price and returns stealth visual indicators.
func (c *MarginController) CheckMargin(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.MarginCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CheckMargin(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Margin evaluation completed", res)
}

// ListHotkeys returns the active fast-grid buttons for cashier counter.
func (c *MarginController) ListHotkeys(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	timeSlot := strings.TrimSpace(r.URL.Query().Get("time_slot"))
	res, err := c.service.ListHotkeys(r.Context(), claims.UserID, timeSlot)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Fast-grid hotkeys retrieved successfully", res)
}

// CreateHotkey adds a new quick tile button.
func (c *MarginController) CreateHotkey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateShopHotkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	hotkey, err := c.service.CreateHotkey(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Fast-grid hotkey created successfully", hotkey)
}

// DeleteHotkey removes a tile button.
func (c *MarginController) DeleteHotkey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	hotkeyID := chi.URLParam(r, "id")
	if strings.TrimSpace(hotkeyID) == "" {
		reuse.Error(w, http.StatusBadRequest, "hotkey id is required")
		return
	}

	if err := c.service.DeleteHotkey(r.Context(), claims.UserID, hotkeyID); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Fast-grid hotkey deleted successfully", nil)
}

// AutoPopulateHotkeys automatically populates tiles from top selling products in the last 14 days.
func (c *MarginController) AutoPopulateHotkeys(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.AutoPopulateHotkeys(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Fast-grid tiles auto-populated from top selling items", res)
}

