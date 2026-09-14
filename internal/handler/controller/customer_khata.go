// Package controller handles customer khata endpoints.
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

type CustomerKhataController struct {
	service *services.KhataService
}

func NewCustomerKhataController(service *services.KhataService) *CustomerKhataController {
	return &CustomerKhataController{
		service: service,
	}
}

// GetCustomerKhataSummary returns the customer's total multi-shop credit portfolio (Protected - Customer).
func (c *CustomerKhataController) GetCustomerKhataSummary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := c.service.GetCustomerKhataSummary(r.Context(), claims.UserID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Customer khata portfolio retrieved successfully", summary)
}

// GetCustomerKhataPassbook returns the detailed transaction passbook for a specific shop (Protected - Customer).
func (c *CustomerKhataController) GetCustomerKhataPassbook(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	passbook, err := c.service.GetCustomerKhataPassbook(r.Context(), claims.UserID, khataID)
	if err != nil {
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Customer passbook retrieved successfully", passbook)
}

// DisputeTransaction flags an incorrect credit entry with a reason (Protected - Customer).
func (c *CustomerKhataController) DisputeTransaction(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	var input dto.DisputeTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.DisputeTransaction(r.Context(), claims.UserID, khataID, input)
	if err != nil {
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata account not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Transaction discrepancy reported to shopkeeper successfully. Entry marked as DISPUTED.", nil)
}

// SubmitUPIPayment records customer direct UPI settlement proof (Protected - Customer).
func (c *CustomerKhataController) SubmitUPIPayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	var input dto.SubmitUPIPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := c.service.SubmitUPIPayment(r.Context(), claims.UserID, khataID, input)
	if err != nil {
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata account not found")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "UPI payment recorded and balance settled successfully", tx)
}

// DownloadCustomerPDF downloads the official statement PDF for the customer (Protected - Customer).
func (c *CustomerKhataController) DownloadCustomerPDF(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	pdfBytes, err := c.service.GenerateCustomerStatementPDF(r.Context(), claims.UserID, khataID)
	if err != nil {
		if errors.Is(err, services.ErrKhataCustomerNotFound) {
			reuse.Error(w, http.StatusNotFound, "khata account not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"khata_statement_%s.pdf\"", khataID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// RequestClosure initiates a dual-OTP closure request by the customer (Protected - Customer)
func (c *CustomerKhataController) RequestClosure(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	res, err := c.service.RequestKhataClosureByCustomer(r.Context(), claims.UserID, khataID)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, res.Message, res)
}

// VerifyClosureOTP allows customer to verify OTP if requested by shopkeeper (Protected - Customer)
func (c *CustomerKhataController) VerifyClosureOTP(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	var input dto.VerifyKhataClosureOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.VerifyKhataClosureOTPByCustomer(r.Context(), claims.UserID, khataID, input.OTP)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Khata successfully closed via mutual Dual-OTP verification", nil)
}

// SetCreditOTPProtection configures customer OTP approval requirement for high-value credit (Protected - Customer)
func (c *CustomerKhataController) SetCreditOTPProtection(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	var input dto.SetCreditOTPProtectionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.SetCreditOTPProtection(r.Context(), claims.UserID, khataID, input.Required, input.Threshold)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Credit OTP protection settings updated successfully", nil)
}

// SetPromiseToPay allows customer to set or confirm their repayment target (Protected - Customer)
func (c *CustomerKhataController) SetPromiseToPay(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	khataID := strings.TrimSpace(chi.URLParam(r, "khataId"))
	if khataID == "" {
		reuse.Error(w, http.StatusBadRequest, "khata id is required")
		return
	}

	var input dto.SetPromiseToPayRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	err := c.service.SetPromiseToPayByCustomer(r.Context(), claims.UserID, khataID, input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Repayment promise target updated successfully", nil)
}


