// Package controller handles HTTP requests for company damage and expiry claims.
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

// DamageClaimController handles claim recording and statements.
type DamageClaimController struct {
	service *services.DamageClaimService
}

// NewDamageClaimController initializes DamageClaimController.
func NewDamageClaimController(service *services.DamageClaimService) *DamageClaimController {
	return &DamageClaimController{
		service: service,
	}
}

// CreateClaim records an unsellable, damaged, or expired item.
func (c *DamageClaimController) CreateClaim(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateDamageClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CreateClaim(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Damage/expiry claim recorded in vault", res)
}

// UpdateClaimStatus marks a claim as resolved (replaced, credit note, or rejected).
func (c *DamageClaimController) UpdateClaimStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	claimID := chi.URLParam(r, "id")
	if claimID == "" {
		reuse.Error(w, http.StatusBadRequest, "claim id is required")
		return
	}

	var input dto.UpdateClaimStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.UpdateClaimStatus(r.Context(), claims.UserID, claimID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Damage claim status updated successfully", res)
}

// ListClaims lists past and pending claims.
func (c *DamageClaimController) ListClaims(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	distributor := r.URL.Query().Get("distributor_name")
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := c.service.ListClaims(r.Context(), claims.UserID, distributor, status, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Damage claims retrieved", res)
}

// GetDistributorStatement produces an aggregated summary sheet when the distributor SR visits.
func (c *DamageClaimController) GetDistributorStatement(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	distributor := r.URL.Query().Get("distributor_name")

	res, err := c.service.GetDistributorStatement(r.Context(), claims.UserID, distributor)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Distributor damage statement retrieved", res)
}

