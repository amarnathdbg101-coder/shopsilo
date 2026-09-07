// Package controller handles incoming HTTP requests.
package controller

import (
	"encoding/json"
	"errors"
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
