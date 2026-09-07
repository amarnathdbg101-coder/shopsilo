// Package controller handles http work.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ReservationController struct {
	service *services.ReservationService
}

func NewReservationController(service *services.ReservationService) *ReservationController {
	return &ReservationController{
		service: service,
	}
}

// Create handles customer requesting an in-store hold on a product (Protected)
func (c *ReservationController) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := c.service.CreateReservation(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrMaxActiveReservations) {
			reuse.Error(w, http.StatusTooManyRequests, err.Error())
			return
		}
		if errors.Is(err, services.ErrProductOutOfStock) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "Item reserved successfully for in-store pickup. Show your 6-digit code at the billing counter.", res)
}

// ListUserReservations returns the logged-in customer's reservations (Protected)
func (c *ReservationController) ListUserReservations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	status := query.Get("status")

	filter := dto.ReservationFilter{
		Status: status,
		Page:   page,
		Limit:  limit,
	}

	res, err := c.service.ListUserReservations(r.Context(), claims.UserID, filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Reservations retrieved successfully", res)
}

// GetByID returns details for a specific reservation (Protected)
func (c *ReservationController) GetByID(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "reservation id is required")
		return
	}

	res, err := c.service.GetReservationByID(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrReservationNotFound) {
			reuse.Error(w, http.StatusNotFound, "reservation not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Reservation details retrieved successfully", res)
}

// CancelUserReservation lets the customer cancel an active hold (Protected)
func (c *ReservationController) CancelUserReservation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "reservation id is required")
		return
	}

	res, err := c.service.CancelUserReservation(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrReservationNotFound) {
			reuse.Error(w, http.StatusNotFound, "active reservation not found or already processed")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Reservation cancelled successfully and stock released", res)
}

// ListShopReservations lets the shop owner view customer holds for their shop (Protected - Shop)
func (c *ReservationController) ListShopReservations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	status := query.Get("status")

	filter := dto.ReservationFilter{
		Status: status,
		Page:   page,
		Limit:  limit,
	}

	res, err := c.service.ListShopReservations(r.Context(), claims.UserID, filter)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop incoming reservations retrieved successfully", res)
}

// VerifyShopReservation validates customer's 6-digit code at store counter (Protected - Shop)
func (c *ReservationController) VerifyShopReservation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.VerifyReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	completed, err := c.service.VerifyShopReservation(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrInvalidVerificationCode) {
			reuse.Error(w, http.StatusBadRequest, "invalid or expired pickup code / reservation number")
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Customer pickup code verified successfully. Sale completed and inventory deducted.", completed)
}

// CancelShopReservation lets the shopkeeper cancel an in-store reservation (Protected - Shop)
func (c *ReservationController) CancelShopReservation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "reservation id is required")
		return
	}

	res, err := c.service.CancelShopReservation(r.Context(), id, claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		if errors.Is(err, services.ErrReservationNotFound) {
			reuse.Error(w, http.StatusNotFound, "active reservation not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Reservation cancelled from shop side and inventory released", res)
}
