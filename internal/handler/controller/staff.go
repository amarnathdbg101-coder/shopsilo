// Package controller handles http work.
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

	"github.com/go-chi/chi/v5"
)

type StaffController struct {
	service *services.StaffService
}

func NewStaffController(service *services.StaffService) *StaffController {
	return &StaffController{
		service: service,
	}
}

// CreateStaff adds a new cashier or helper sub-account (Protected - Shop Owner)
func (c *StaffController) CreateStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	staff, err := c.service.CreateStaff(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrStaffPhoneExists) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Staff sub-account created successfully", staff)
}

// ListStaff returns all cashier/helper sub-accounts working in the shop (Protected - Shop Owner)
func (c *StaffController) ListStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := c.service.ListStaff(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop staff list retrieved successfully", list)
}

// UpdateStaff updates staff info or 4-digit PIN (Protected - Shop Owner)
func (c *StaffController) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	staffID := chi.URLParam(r, "id")
	if staffID == "" {
		reuse.Error(w, http.StatusBadRequest, "staff id is required")
		return
	}

	var input dto.UpdateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := c.service.UpdateStaff(r.Context(), claims.UserID, staffID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrStaffNotFound) {
			reuse.Error(w, http.StatusNotFound, "staff member not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Staff details updated successfully", updated)
}

// DeleteStaff removes a cashier sub-account (Protected - Shop Owner)
func (c *StaffController) DeleteStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	staffID := chi.URLParam(r, "id")
	if staffID == "" {
		reuse.Error(w, http.StatusBadRequest, "staff id is required")
		return
	}

	err := c.service.DeleteStaff(r.Context(), claims.UserID, staffID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrStaffNotFound) {
			reuse.Error(w, http.StatusNotFound, "staff member not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Staff sub-account deleted successfully", nil)
}

// StaffLogin authenticates a cashier/helper using 4-digit PIN (Public)
func (c *StaffController) StaffLogin(w http.ResponseWriter, r *http.Request) {
	var input dto.StaffLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.StaffLogin(r.Context(), input)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidStaffPIN) {
			reuse.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, repository.ErrStaffDeactivated) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Cashier login successful", res)
}
