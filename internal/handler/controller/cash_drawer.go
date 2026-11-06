// Package controller handles HTTP requests for cash drawer telemetry and shift tracking.
package controller

import (
	"encoding/json"
	"net/http"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

// CashDrawerController handles shift opening/closing and drawer telemetry.
type CashDrawerController struct {
	service *services.CashDrawerService
}

// NewCashDrawerController creates a new CashDrawerController.
func NewCashDrawerController(service *services.CashDrawerService) *CashDrawerController {
	return &CashDrawerController{
		service: service,
	}
}

// GetCurrentStatus returns the active shift drawer metrics and safe drop status.
func (c *CashDrawerController) GetCurrentStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetCurrentStatus(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Current drawer status retrieved successfully", res)
}

// OpenSession begins a new shift.
func (c *CashDrawerController) OpenSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.OpenDrawerSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := c.service.OpenSession(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Cash drawer shift opened successfully", sess)
}

// RecordSafeDrop deposits cash into safe vault.
func (c *CashDrawerController) RecordSafeDrop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.RecordSafeDropRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := c.service.RecordSafeDrop(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Safe drop recorded successfully", sess)
}

// RecordPettyExpense logs store operational petty cash expense.
func (c *CashDrawerController) RecordPettyExpense(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.RecordPettyExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := c.service.RecordPettyExpense(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Petty cash expense recorded successfully", sess)
}

// CloseSession reconciles actual cash and closes shift.
func (c *CashDrawerController) CloseSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CloseDrawerSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := c.service.CloseSession(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Cash drawer shift closed and reconciled successfully", sess)
}

// ListHistory retrieves past drawer shifts.
func (c *CashDrawerController) ListHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.ListHistory(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Drawer shift history retrieved successfully", res)
}

