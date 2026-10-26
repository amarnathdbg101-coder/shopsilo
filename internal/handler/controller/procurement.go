// Package controller handles http request processing.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

type ProcurementController struct {
	service *services.ProcurementService
}

func NewProcurementController(service *services.ProcurementService) *ProcurementController {
	return &ProcurementController{service: service}
}

// List returns all procurement items for the authenticated merchant's shop.
func (c *ProcurementController) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := c.service.ListProcurementItems(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Procurement items retrieved successfully", items)
}

// Create adds a new item to the shop's procurement list.
func (c *ProcurementController) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateProcurementItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		reuse.Error(w, http.StatusBadRequest, "item name is required")
		return
	}

	created, err := c.service.CreateProcurementItem(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Procurement item created successfully", created)
}

// Update modifies an existing procurement item.
func (c *ProcurementController) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "item id is required")
		return
	}

	var req dto.UpdateProcurementItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := c.service.UpdateProcurementItem(r.Context(), claims.UserID, id, req)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrProcurementItemNotFound) {
			reuse.Error(w, http.StatusNotFound, "procurement item not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Procurement item updated successfully", updated)
}

// Toggle flips the is_completed checkbox.
func (c *ProcurementController) Toggle(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "item id is required")
		return
	}

	toggled, err := c.service.ToggleProcurementItem(r.Context(), claims.UserID, id)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Procurement item toggled successfully", toggled)
}

// Delete removes an item from the procurement list.
func (c *ProcurementController) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "item id is required")
		return
	}

	if err := c.service.DeleteProcurementItem(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrProcurementItemNotFound) {
			reuse.Error(w, http.StatusNotFound, "procurement item not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Procurement item deleted successfully", nil)
}

// ClearCompleted removes all completed items from the procurement list.
func (c *ProcurementController) ClearCompleted(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := c.service.ClearCompleted(r.Context(), claims.UserID); err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Completed procurement items cleared successfully", nil)
}
