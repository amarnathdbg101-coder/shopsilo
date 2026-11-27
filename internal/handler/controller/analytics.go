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

func parseDateRangeOrMonth(r *http.Request) (time.Time, time.Time) {
	q := r.URL.Query()
	now := time.Now()
	if fromStr, toStr := q.Get("from"), q.Get("to"); fromStr != "" && toStr != "" {
		if f, err1 := time.Parse("2006-01-02", fromStr); err1 == nil {
			if t, err2 := time.Parse("2006-01-02", toStr); err2 == nil && !t.Before(f) {
				return f, t
			}
		}
	}
	year, _ := strconv.Atoi(q.Get("year"))
	month, _ := strconv.Atoi(q.Get("month"))
	if year <= 0 {
		year = now.Year()
	}
	if month <= 0 || month > 12 {
		month = int(now.Month())
	}
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 1, -1)
	return from, to
}

// GetShopIntelligence returns GMROI, Inventory Turnover, ABC-XYZ, Market Basket Lift, Peak Hours & Tender Reconciliation.
func (c *AnalyticsController) GetShopIntelligence(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fromDate, toDate := parseDateRangeOrMonth(r)
	res, err := c.service.GetShopIntelligence(r.Context(), claims.UserID, fromDate, toDate)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop intelligence analytics retrieved successfully", res)
}

// DownloadCACSV streams Tally/Excel-compatible UTF-8 BOM CSV for CA audits (sales, expenses, stock, khata).
func (c *AnalyticsController) DownloadCACSV(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "sales"
	}
	fromDate, toDate := parseDateRangeOrMonth(r)

	csvBytes, filename, err := c.service.ExportCACSV(r.Context(), claims.UserID, exportType, fromDate, toDate)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBytes)
}

// GetDukaanPulse returns a pre-digested executive summary with 7-day trend, health score, and top actionable insights.
func (c *AnalyticsController) GetDukaanPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireUserClaims(w, r)
	if !ok {
		return
	}

	res, err := c.service.GetDukaanPulse(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Dukaan pulse retrieved successfully", res)
}

// GetBusinessAdvisor returns CA-grade product decisions (what sells, why, how much to stock, and what to drop).
func (c *AnalyticsController) GetBusinessAdvisor(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireUserClaims(w, r)
	if !ok {
		return
	}

	res, err := c.service.GetBusinessAdvisor(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Business advisor report retrieved successfully", res)
}

// GetEveningDigest returns today's cashflow split, net profit, top reorders, and WhatsApp-ready digest text.
func (c *AnalyticsController) GetEveningDigest(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.RequireUserClaims(w, r)
	if !ok {
		return
	}

	lang := r.URL.Query().Get("lang")
	res, err := c.service.GetEveningDigest(r.Context(), claims.UserID, lang)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Evening digest retrieved successfully", res)
}
