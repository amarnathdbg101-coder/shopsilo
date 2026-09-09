// Package controller handles incoming HTTP requests.
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strings"

	"github.com/go-chi/chi/v5"
)

type KhataController struct {
	service *services.KhataService
}

func NewKhataController(service *services.KhataService) *KhataController {
	return &KhataController{
		service: service,
	}
}

// RecordCredit gives udhar / adds credit to a customer's ledger (Protected - Shop)
func (c *KhataController) RecordCredit(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.RecordCreditTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := c.service.RecordCredit(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Credit (udhar) recorded successfully in khata", tx)
}

// RecordPayment records customer payment settlement / jama (Protected - Shop)
func (c *KhataController) RecordPayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	var input dto.RecordPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := c.service.RecordPayment(r.Context(), claims.UserID, customerMobile, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Payment (jama) recorded and balance settled successfully", tx)
}

// ListCustomers returns all khata accounts with balances (Protected - Shop)
func (c *KhataController) ListCustomers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	search := r.URL.Query().Get("search")
	onlyWithBalance := r.URL.Query().Get("only_balance") == "true"

	customers, err := c.service.ListCustomers(r.Context(), claims.UserID, search, onlyWithBalance)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Khata customers retrieved successfully", customers)
}

// GetCustomerHistory returns a customer's detailed passbook ledger (Protected - Shop)
func (c *KhataController) GetCustomerHistory(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	history, err := c.service.GetCustomerHistory(r.Context(), claims.UserID, customerMobile)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata customer account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Customer khata passbook retrieved successfully", history)
}

// GetSummary returns total market udhar and customer count (Protected - Shop)
func (c *KhataController) GetSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := c.service.GetSummary(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Khata summary retrieved successfully", summary)
}

// GetPaymentReminder generates a WhatsApp Click-to-Chat reminder link for an udhar customer (Protected - Shop)
func (c *KhataController) GetPaymentReminder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	reminder, err := c.service.GeneratePaymentReminder(r.Context(), claims.UserID, customerMobile)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata customer account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "WhatsApp payment reminder generated successfully", reminder)
}

// UpdateCreditLimit updates the maximum allowed credit cap for a customer (Protected - Shop)
func (c *KhataController) UpdateCreditLimit(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	var input dto.SetCreditLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.UpdateCreditLimit(r.Context(), claims.UserID, customerMobile, input.CreditLimit)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata customer account not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Customer credit limit updated successfully", map[string]interface{}{
		"customer_mobile": customerMobile,
		"credit_limit":    input.CreditLimit,
	})
}

// GetAgingReport calculates bad-debt aging brackets and overdue accounts (Protected - Shop)
func (c *KhataController) GetAgingReport(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	report, err := c.service.GetAgingReport(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Khata bad debt aging report generated successfully", report)
}

// DownloadStatementPDF downloads an itemized account statement PDF for a customer (Protected - Shop)
func (c *KhataController) DownloadStatementPDF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	customerMobile = strings.TrimSuffix(customerMobile, ".pdf")
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	pdfBytes, err := c.service.GenerateStatementPDF(r.Context(), claims.UserID, customerMobile)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata customer account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"khata_statement_%s.pdf\"", customerMobile))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// GetStatementShare generates a WhatsApp passbook link with summary text (Protected - Shop)
func (c *KhataController) GetStatementShare(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	customerMobile := strings.TrimSpace(chi.URLParam(r, "mobile"))
	if customerMobile == "" {
		reuse.Error(w, http.StatusBadRequest, "customer mobile is required")
		return
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	res, err := c.service.GetStatementShareLink(r.Context(), claims.UserID, customerMobile, baseURL)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata customer account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "WhatsApp statement link generated successfully", res)
}

