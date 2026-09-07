// Package controller handles incoming HTTP requests.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type ExpenseController struct {
	service *services.ExpenseService
}

func NewExpenseController(service *services.ExpenseService) *ExpenseController {
	return &ExpenseController{
		service: service,
	}
}

// CreateExpense records a daily store expense (Protected - Shop)
func (c *ExpenseController) CreateExpense(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	exp, err := c.service.CreateExpense(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Store expense recorded successfully", exp)
}

// ListExpenses lists store expenses for a month (Protected - Shop)
func (c *ExpenseController) ListExpenses(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	category := r.URL.Query().Get("category")

	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)

	if year <= 0 {
		year = time.Now().Year()
	}
	if month <= 0 {
		month = int(time.Now().Month())
	}

	expenses, err := c.service.ListExpenses(r.Context(), claims.UserID, year, month, category)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Store expenses retrieved successfully", expenses)
}

// DeleteExpense removes an expense entry (Protected - Shop)
func (c *ExpenseController) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	expenseID := strings.TrimSpace(chi.URLParam(r, "id"))
	if expenseID == "" {
		reuse.Error(w, http.StatusBadRequest, "expense id is required")
		return
	}

	err := c.service.DeleteExpense(r.Context(), claims.UserID, expenseID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, repository.ErrExpenseNotFound) {
			reuse.Error(w, http.StatusNotFound, "expense not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Store expense deleted successfully", nil)
}
