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

type ReviewController struct {
	service *services.ReviewService
}

func NewReviewController(service *services.ReviewService) *ReviewController {
	return &ReviewController{
		service: service,
	}
}

// AddOrUpdate creates or updates a review for a shop (Protected)
func (c *ReviewController) AddOrUpdate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	var input dto.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	rev, err := c.service.AddOrUpdateReview(r.Context(), slug, claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, services.ErrSelfReviewNotAllowed) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Success(w, "Review submitted successfully", rev)
}

// List returns public reviews and rating breakdown for a shop (Public)
func (c *ReviewController) List(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	res, err := c.service.ListShopReviews(r.Context(), slug, page, limit)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop reviews retrieved successfully", res)
}

// Delete removes the user's review for a shop (Protected)
func (c *ReviewController) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	err := c.service.DeleteMyReview(r.Context(), slug, claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) || errors.Is(err, services.ErrReservationNotFound) {
			reuse.Error(w, http.StatusNotFound, "review or shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Review deleted successfully", nil)
}
