// Package controller handles HTTP requests for End-of-Day P&L PDF reports.
package controller

import (
	"fmt"
	"net/http"
	"time"

	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

// EODReportController handles End-of-Day P&L statement generation and PDF downloads.
type EODReportController struct {
	pdfService *services.EODPDFService
}

// NewEODReportController initializes EODReportController.
func NewEODReportController(pdfService *services.EODPDFService) *EODReportController {
	return &EODReportController{
		pdfService: pdfService,
	}
}

// DownloadEndOfDayPLPDF generates and streams an executive P&L PDF report using high-performance pure Go gopdf engine.
func (c *EODReportController) DownloadEndOfDayPLPDF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = parsed
		}
	}

	pdfBytes, err := c.pdfService.GenerateEndOfDayPLPDF(r.Context(), claims.UserID, targetDate)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate PDF report: %v", err))
		return
	}

	filename := fmt.Sprintf("EOD-PL-%s.pdf", targetDate.Format("20060102"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

