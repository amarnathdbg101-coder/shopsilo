package controller

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type TelemetryController struct {
	service *services.TelemetryService
}

func NewTelemetryController(service *services.TelemetryService) *TelemetryController {
	return &TelemetryController{service: service}
}

// LogFrontendError receives error telemetry from frontend (Public Interceptor)
func (c *TelemetryController) LogFrontendError(w http.ResponseWriter, r *http.Request) {
	var req dto.TelemetryErrorRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil {
		req.UserID = claims.UserID
		req.UserRole = claims.Role
	}

	_ = c.service.RecordError(r.Context(), req)
	reuse.Success(w, "Telemetry recorded", nil)
}

// GetAdminErrors retrieves active 24-hour developer error log list (Protected - Admin)
func (c *TelemetryController) GetAdminErrors(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		reuse.Error(w, http.StatusUnauthorized, "admin access required")
		return
	}

	records, err := c.service.Get24HourErrors(r.Context())
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "24-hour developer error logs retrieved", records)
}

// ClearAdminErrors purges error log buffer (Protected - Admin)
func (c *TelemetryController) ClearAdminErrors(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		reuse.Error(w, http.StatusUnauthorized, "admin access required")
		return
	}

	_ = c.service.ClearErrors(r.Context())
	reuse.Success(w, "Developer error logs cleared", nil)
}
