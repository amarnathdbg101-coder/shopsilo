// Package controller handles HTTP requests for no-bill reverse return lookups.
package controller

import (
	"net/http"
	"strconv"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

// ReturnLookupController handles finding previous sales for customer returns.
type ReturnLookupController struct {
	service *services.ReturnLookupService
}

// NewReturnLookupController initializes ReturnLookupController.
func NewReturnLookupController(service *services.ReturnLookupService) *ReturnLookupController {
	return &ReturnLookupController{
		service: service,
	}
}

// LookupReturn searches past bills within N days by customer phone, barcode, or product ID.
func (c *ReturnLookupController) LookupReturn(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	filter := dto.ReturnLookupFilter{
		CustomerPhone: r.URL.Query().Get("customer_phone"),
		Barcode:       r.URL.Query().Get("barcode"),
		ProductID:     r.URL.Query().Get("product_id"),
		BillNumber:    r.URL.Query().Get("bill_number"),
		Days:          days,
	}

	res, err := c.service.LookupReturn(r.Context(), claims.UserID, filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Past purchases found for return verification", res)
}

