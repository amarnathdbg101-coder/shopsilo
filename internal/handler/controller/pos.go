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
	"strconv"
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

	if input.PaymentMethod == "split" && input.SplitPayments == nil {
		reuse.Error(w, http.StatusBadRequest, "split_payments details required when payment method is split")
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

// ScanBarcode looks up a product by barcode/SKU during POS checkout (Protected - Shop)
func (c *POSController) ScanBarcode(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sku := chi.URLParam(r, "sku")
	if strings.TrimSpace(sku) == "" {
		reuse.Error(w, http.StatusBadRequest, "barcode / SKU is required")
		return
	}

	res, err := c.service.ScanBarcode(r.Context(), claims.UserID, sku)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Barcode scanned successfully", res)
}

// ShareBill generates a WhatsApp Click-to-Chat URL for sending the bill receipt to customer (Protected - Shop)
func (c *POSController) ShareBill(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	billNumber := chi.URLParam(r, "bill_number")
	billNumber = strings.TrimSuffix(billNumber, ".pdf")
	if strings.TrimSpace(billNumber) == "" {
		reuse.Error(w, http.StatusBadRequest, "bill number is required")
		return
	}

	res, err := c.service.GetBillShareLink(r.Context(), claims.UserID, billNumber)
	if err != nil {
		if errors.Is(err, repository.ErrBillNotFound) {
			reuse.Error(w, http.StatusNotFound, "bill not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "WhatsApp receipt share link generated", res)
}

// GetDailyCloseReport returns the end-of-day drawer cash reconciliation report (Protected - Shop)
func (c *POSController) GetDailyCloseReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dateStr := r.URL.Query().Get("date")
	report, err := c.service.GetDailyCloseReport(r.Context(), claims.UserID, dateStr)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Daily cash drawer closing report generated", report)
}

// GetMonthlyGSTReport calculates monthly taxable turnover, CGST, and SGST for CA filing (Protected - Shop)
func (c *POSController) GetMonthlyGSTReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	taxRate, _ := strconv.ParseFloat(r.URL.Query().Get("tax_rate"), 64)

	report, err := c.service.GetMonthlyGSTReport(r.Context(), claims.UserID, year, month, taxRate)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Monthly GST report generated successfully", report)
}

// ParkBill temporarily holds a counter cart when a customer steps away (Protected - Shop)
func (c *POSController) ParkBill(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.ParkPOSBillRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	parked, err := c.service.ParkBill(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Counter cart parked successfully", parked)
}

// ListParkedBills returns all carts currently on hold (Protected - Shop)
func (c *POSController) ListParkedBills(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := c.service.ListParkedBills(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Parked carts retrieved successfully", list)
}

// GetParkedBill retrieves a held cart so the register can resume billing (Protected - Shop)
func (c *POSController) GetParkedBill(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		reuse.Error(w, http.StatusBadRequest, "parked cart id is required")
		return
	}

	cart, err := c.service.GetParkedBill(r.Context(), claims.UserID, id)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrParkedBillNotFound) {
			reuse.Error(w, http.StatusNotFound, "parked cart not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Parked cart retrieved successfully", cart)
}

// DeleteParkedBill discards or finalizes a held cart (Protected - Shop)
func (c *POSController) DeleteParkedBill(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		reuse.Error(w, http.StatusBadRequest, "parked cart id is required")
		return
	}

	err := c.service.DeleteParkedBill(r.Context(), claims.UserID, id)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrParkedBillNotFound) {
			reuse.Error(w, http.StatusNotFound, "parked cart not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Parked cart removed successfully", nil)
}

// ParseParchi converts raw pasted grocery text or WhatsApp messages into a ready-to-bill POS cart (Protected - Shop)
func (c *POSController) ParseParchi(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ParseParchiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.ParseParchi(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Parchi parsed successfully into draft POS cart", res)
}

// GetWeeklyScorecard returns an executive business health and net khata cash flow summary (Protected - Shop)
func (c *POSController) GetWeeklyScorecard(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := c.service.GetWeeklyScorecard(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Weekly business scorecard generated successfully", res)
}

// GetCustomerRecentBasket retrieves the customer's previous grocery purchase to enable 1-click repeat reordering (Protected - Shop)
func (c *POSController) GetCustomerRecentBasket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	phone := chi.URLParam(r, "phone")
	if strings.TrimSpace(phone) == "" {
		reuse.Error(w, http.StatusBadRequest, "customer phone number is required")
		return
	}

	res, err := c.service.GetCustomerRecentBasket(r.Context(), claims.UserID, phone)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusNotFound, err.Error())
		return
	}

	reuse.Success(w, "Customer recent purchase basket retrieved successfully", res)
}

// ProcessPOSReturn handles counter returns against an existing POS bill (Protected - Shop)
func (c *POSController) ProcessPOSReturn(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ProcessPOSReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.ProcessPOSReturn(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "POS return processed and inventory restocked successfully", res)
}




