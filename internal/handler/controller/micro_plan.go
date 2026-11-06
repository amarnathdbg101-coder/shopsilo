// Package controller handles HTTP requests for khata micro-installment debt recovery.
package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

// MicroPlanController handles customer micro-repayment endpoints.
type MicroPlanController struct {
	service *services.MicroPlanService
}

// NewMicroPlanController initializes MicroPlanController.
func NewMicroPlanController(service *services.MicroPlanService) *MicroPlanController {
	return &MicroPlanController{
		service: service,
	}
}

// CreatePlan initializes a new daily/weekly micro-installment plan.
func (c *MicroPlanController) CreatePlan(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateMicroPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CreatePlan(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Khata micro-installment plan created", res)
}

// RecordPayment logs a single installment payment.
func (c *MicroPlanController) RecordPayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	planID := chi.URLParam(r, "id")
	if planID == "" {
		reuse.Error(w, http.StatusBadRequest, "plan id is required")
		return
	}

	var input dto.RecordMicroPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.RecordPayment(r.Context(), claims.UserID, planID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Micro installment payment recorded", res)
}

// GetPlanSummary returns full plan metrics, progress, and payment logs.
func (c *MicroPlanController) GetPlanSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	planID := chi.URLParam(r, "id")
	if planID == "" {
		reuse.Error(w, http.StatusBadRequest, "plan id is required")
		return
	}

	res, err := c.service.GetPlanSummary(r.Context(), claims.UserID, planID)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Micro plan summary retrieved", res)
}

// ListPlans lists micro repayment plans for a shop.
func (c *MicroPlanController) ListPlans(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := c.service.ListPlans(r.Context(), claims.UserID, status, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Micro repayment plans retrieved", res)
}

