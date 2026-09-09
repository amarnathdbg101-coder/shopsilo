// Package controller handles HTTP requests.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ModerationController struct {
	service *services.ModerationService
}

func NewModerationController(service *services.ModerationService) *ModerationController {
	return &ModerationController{
		service: service,
	}
}

// SubmitReport handles reporting a shop, product, or review (Public / Authenticated)
func (c *ModerationController) SubmitReport(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var reporterUserID *string
	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil && claims.UserID != "" {
		reporterUserID = &claims.UserID
	}

	ip := middleware.ExtractClientIP(r)
	userAgent := r.UserAgent()

	report, quarantined, err := c.service.SubmitReport(r.Context(), reporterUserID, input, ip, userAgent)
	if err != nil {
		if errors.Is(err, services.ErrTargetNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	message := "Report submitted successfully. Our safety team will review it."
	if quarantined {
		message = "Report submitted. Target has been temporarily quarantined pending immediate review under IT Rules safety guidelines."
	}

	reuse.Created(w, message, map[string]interface{}{
		"report":      report,
		"quarantined": quarantined,
	})
}

// ListReports lists submitted reports (Protected - Admin)
func (c *ModerationController) ListReports(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	reports, err := c.service.ListReports(r.Context(), status, page, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Reports retrieved successfully", reports)
}

// ResolveReport marks a report as reviewed, dismissed, or action taken (Protected - Admin)
func (c *ModerationController) ResolveReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "id")
	if reportID == "" {
		reuse.Error(w, http.StatusBadRequest, "report id is required")
		return
	}

	var input dto.ResolveReportRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	adminID := ""
	if claims != nil {
		adminID = claims.UserID
	}

	if err := c.service.ResolveReport(r.Context(), reportID, input, adminID); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Report resolved successfully", nil)
}

// BanShop performs 1-click ban on shop & malicious user (Protected - Admin)
func (c *ModerationController) BanShop(w http.ResponseWriter, r *http.Request) {
	shopID := chi.URLParam(r, "id")
	if shopID == "" {
		reuse.Error(w, http.StatusBadRequest, "shop id is required")
		return
	}

	var input dto.BanShopRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	adminID := ""
	if claims != nil {
		adminID = claims.UserID
	}

	if err := c.service.BanShop(r.Context(), shopID, input, adminID); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop banned and associated identifiers blacklisted successfully", nil)
}

// ListBannedEntities lists all blacklist entries (Protected - Admin)
func (c *ModerationController) ListBannedEntities(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	banned, err := c.service.ListBannedEntities(r.Context(), page, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Banned entities retrieved successfully", banned)
}

// AddBannedEntity adds a new blacklist entry (Protected - Admin)
func (c *ModerationController) AddBannedEntity(w http.ResponseWriter, r *http.Request) {
	var input dto.AddBannedEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	adminID := ""
	if claims != nil {
		adminID = claims.UserID
	}

	if err := c.service.AddBannedEntity(r.Context(), input, adminID); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Entity added to blacklist successfully", nil)
}

// UnbanEntity removes an entity from blacklist (Protected - Admin)
func (c *ModerationController) UnbanEntity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := c.service.UnbanEntity(r.Context(), id); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Entity unbanned successfully", nil)
}

// GetStats returns key operational metrics for the admin console (Protected - Admin)
func (c *ModerationController) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.service.GetAdminStats(r.Context())
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Admin stats retrieved successfully", stats)
}

// ListAdminShops lists all shops with filtering by status and search (Protected - Admin)
func (c *ModerationController) ListAdminShops(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	shops, total, err := c.service.ListShopsForAdmin(r.Context(), status, search, page, limit)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Admin shops retrieved successfully", map[string]interface{}{
		"shops": shops,
		"total": total,
	})
}

// UpdateAdminShopStatus updates a shop's status (Approve, Suspend, etc.) (Protected - Admin)
func (c *ModerationController) UpdateAdminShopStatus(w http.ResponseWriter, r *http.Request) {
	shopID := chi.URLParam(r, "id")
	if shopID == "" {
		reuse.Error(w, http.StatusBadRequest, "shop id is required")
		return
	}

	var input dto.UpdateShopStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	adminID := ""
	if claims != nil {
		adminID = claims.UserID
	}

	if err := c.service.UpdateShopStatus(r.Context(), shopID, input.Status, input.Reason, adminID); err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop status updated successfully", map[string]string{
		"shop_id": shopID,
		"status":  input.Status,
	})
}
