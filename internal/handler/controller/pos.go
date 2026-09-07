// Package controller handles http work.
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strings"

	"github.com/go-chi/chi/v5"
)

type POSController struct {
	service *services.POSService
}

func NewPOSController(service *services.POSService) *POSController {
	return &POSController{
		service: service,
	}
}

// CreateSale records a walk-in counter sale, deducts stock, and generates digital bill (Protected - Shop)
func (c *POSController) CreateSale(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreatePOSSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CreateSale(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Counter sale completed successfully and stock deducted", res)
}

// GetDailySummary returns today's cash & UPI drawer collections and total bills (Protected - Shop)
func (c *POSController) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := c.service.GetDailySummary(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Daily sales summary retrieved successfully", summary)
}

// DownloadReceiptPDF serves digital receipt in PDF format (Protected - Shop)
func (c *POSController) DownloadReceiptPDF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	billNumber := chi.URLParam(r, "bill_number")
	billNumber = strings.TrimSuffix(billNumber, ".pdf")
	if billNumber == "" {
		reuse.Error(w, http.StatusBadRequest, "bill number is required")
		return
	}

	pdfBytes, err := c.service.GenerateReceiptPDF(r.Context(), billNumber)
	if err != nil {
		if errors.Is(err, repository.ErrBillNotFound) {
			reuse.Error(w, http.StatusNotFound, "bill not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="receipt-%s.pdf"`, billNumber))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// ViewPublicReceiptPDF allows customers to view digital receipt via WhatsApp link (Public)
func (c *POSController) ViewPublicReceiptPDF(w http.ResponseWriter, r *http.Request) {
	billNumber := chi.URLParam(r, "bill_number")
	billNumber = strings.TrimSuffix(billNumber, ".pdf")
	if billNumber == "" {
		reuse.Error(w, http.StatusBadRequest, "bill number is required")
		return
	}

	pdfBytes, err := c.service.GenerateReceiptPDF(r.Context(), billNumber)
	if err != nil {
		if errors.Is(err, repository.ErrBillNotFound) {
			reuse.Error(w, http.StatusNotFound, "bill not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="bill-%s.pdf"`, billNumber))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
