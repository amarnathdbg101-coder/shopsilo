// Package controller handles HTTP request processing for the Retail Merchant Co-Pilot.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

type CopilotController struct {
	service *services.CopilotService
}

func NewCopilotController(service *services.CopilotService) *CopilotController {
	return &CopilotController{service: service}
}

// GetDailyPrescriptions returns the store's Morning 3-Action Brief.
func (c *CopilotController) GetDailyPrescriptions(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prescriptions, err := c.service.GetDailyPrescriptions(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Daily prescriptions retrieved successfully", prescriptions)
}

// ExecutePrescription handles the 1-tap execution of an action card.
func (c *CopilotController) ExecutePrescription(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prescriptionID := chi.URLParam(r, "id")
	if strings.TrimSpace(prescriptionID) == "" {
		reuse.Error(w, http.StatusBadRequest, "prescription id is required")
		return
	}

	err := c.service.ExecutePrescription(r.Context(), claims.UserID, prescriptionID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Prescription action executed successfully", map[string]any{"executed": true})
}

// GetDeadCapitalSummary returns aggregated dead capital metrics and monthly carrying cost.
func (c *CopilotController) GetDeadCapitalSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := c.service.GetDeadCapitalSummary(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Dead capital summary retrieved successfully", summary)
}

// EvaluateBargain provides confidential margin-aware negotiation evaluation at the billing counter.
func (c *CopilotController) EvaluateBargain(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.EvaluateBargainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProductID == "" || req.CustomerOfferedPrice <= 0 {
		reuse.Error(w, http.StatusBadRequest, "product_id and a valid customer_offered_price (> 0) are required")
		return
	}

	eval, err := c.service.EvaluateBargain(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Bargain evaluated successfully", eval)
}

// SearchSubstitutes finds exact in-stock items or the best substitutes based on domain attributes.
func (c *CopilotController) SearchSubstitutes(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SearchSubstituteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		reuse.Error(w, http.StatusBadRequest, "query cannot be empty")
		return
	}

	results, err := c.service.SearchSubstitutes(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Substitutes found successfully", results)
}

// Recalculate triggers an on-demand recalculation of ABC-XYZ demand matrix and fresh prescriptions.
func (c *CopilotController) Recalculate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := c.service.RecalculateIntelligence(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopRequired) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Store intelligence recalculated successfully", map[string]any{"recalculated": true})
}
