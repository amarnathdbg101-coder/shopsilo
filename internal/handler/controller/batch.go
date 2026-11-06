// Package controller handles HTTP requests for batch expiry and auto-markdown engines.
package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

// BatchController handles endpoints for batch procurement and expiry radar.
type BatchController struct {
	service *services.BatchService
}

// NewBatchController initializes BatchController.
func NewBatchController(service *services.BatchService) *BatchController {
	return &BatchController{
		service: service,
	}
}

// CreateBatch logs a newly delivered batch with manufacturing and expiry dates.
func (c *BatchController) CreateBatch(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CreateBatch(r.Context(), claims.UserID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Product batch registered successfully", res)
}

// ListNearExpiry finds batches approaching their expiry date.
func (c *BatchController) ListNearExpiry(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	daysWindow, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if daysWindow <= 0 {
		daysWindow = 30
	}

	res, err := c.service.ListNearExpiryBatches(r.Context(), claims.UserID, daysWindow)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Near-expiry batches retrieved", res)
}

// RunMarkdownScan manually or periodically triggers the auto-markdown engine.
func (c *BatchController) RunMarkdownScan(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	daysThreshold, _ := strconv.Atoi(r.URL.Query().Get("threshold_days"))
	defaultDiscount, _ := strconv.ParseFloat(r.URL.Query().Get("discount_percent"), 64)

	res, err := c.service.RunMarkdownScan(r.Context(), claims.UserID, daysThreshold, defaultDiscount)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Auto-markdown engine completed successfully", res)
}

