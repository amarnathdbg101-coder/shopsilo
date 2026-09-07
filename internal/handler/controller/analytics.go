// Package controller handles http work.
package controller

import (
	"errors"
	"net/http"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"
	"time"
)

type AnalyticsController struct {
	service *services.AnalyticsService
}

func NewAnalyticsController(service *services.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{
		service: service,
	}
}

// GetMonthlyProfit returns monthly revenue, wholesale costs, and net profit (Protected - Shop Owner)
func (c *AnalyticsController) GetMonthlyProfit(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	year, _ := strconv.Atoi(query.Get("year"))
	month, _ := strconv.Atoi(query.Get("month"))

	if year <= 0 {
		year = time.Now().Year()
	}
	if month <= 0 || month > 12 {
		month = int(time.Now().Month())
	}

	res, err := c.service.GetMonthlyProfit(r.Context(), claims.UserID, year, month)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Monthly profit analytics retrieved successfully", res)
}

// GetProductMatrix returns Best, Worst, Old Dead Stock, and New Arrival items (Protected - Shop Owner)
func (c *AnalyticsController) GetProductMatrix(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetProductMatrix(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product performance matrix retrieved successfully", res)
}
