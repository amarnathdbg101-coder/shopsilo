// Package controller handler http work.
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
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ShopController struct {
	service *services.ShopService
}

func NewShopController(service *services.ShopService) *ShopController {
	return &ShopController{
		service: service,
	}
}

// Create handles shop onboarding (Protected - Customer/Owner)
func (c *ShopController) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.CreateShopRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	clientIP := middleware.ExtractClientIP(r)
	userAgent := r.UserAgent()
	deviceFP := r.Header.Get("X-Device-Fingerprint")

	res, err := c.service.CreateShop(r.Context(), claims.UserID, claims.Email, clientIP, userAgent, deviceFP, input)
	if err != nil {
		if errors.Is(err, services.ErrRestrictedRegistration) {
			reuse.Error(w, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, services.ErrUserAlreadyHasShop) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, services.ErrSlugAlreadyTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Created(w, "Shop created successfully and account upgraded to shop role", res)
}

// GetMyShop handles retrieving current user's shop (Protected - Owner)
func (c *ShopController) GetMyShop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shop, err := c.service.GetMyShop(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop retrieved successfully", shop)
}

// UpdateMyShop handles updating current user's shop (Protected - Owner)
func (c *ShopController) UpdateMyShop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.UpdateShopRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := reuse.ValidateStruct(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedShop, err := c.service.UpdateMyShop(r.Context(), claims.UserID, input)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		if errors.Is(err, services.ErrSlugAlreadyTaken) {
			reuse.Error(w, http.StatusConflict, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop updated successfully", updatedShop)
}

// ToggleStatus handles toggling shop open/close status (Protected - Owner)
func (c *ShopController) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input dto.ToggleShopStatusRequest
	hasExplicitInput := false
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err == nil {
			hasExplicitInput = true
		}
	}

	targetIsOpen := input.IsOpen
	if !hasExplicitInput {
		// Auto-toggle: retrieve current shop and invert is_open
		currentShop, err := c.service.GetMyShop(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, services.ErrShopNotFound) {
				reuse.Error(w, http.StatusNotFound, "shop not found")
				return
			}
			reuse.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		targetIsOpen = !currentShop.IsOpen
	}

	updatedShop, err := c.service.ToggleShopStatus(r.Context(), claims.UserID, targetIsOpen)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	message := "Shop status set to closed"
	if updatedShop.IsOpen {
		message = "Shop status set to open"
	}

	reuse.Success(w, message, updatedShop)
}

// DeleteMyShop handles closing/deleting shop and reverting role (Protected - Owner)
func (c *ShopController) DeleteMyShop(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	newToken, err := c.service.DeleteMyShop(r.Context(), claims.UserID, claims.Email)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop deleted successfully and account reverted to customer role", map[string]string{
		"access_token": newToken,
	})
}

// List handles searching, filtering by geo location / city / pincode, and paginating shops (Public)
func (c *ShopController) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := query.Get("q")
	if search == "" {
		search = query.Get("search")
	}
	category := query.Get("category")
	city := query.Get("city")
	pincode := query.Get("pincode")

	var lat *float64
	if latStr := query.Get("lat"); latStr != "" {
		if val, err := strconv.ParseFloat(latStr, 64); err == nil {
			lat = &val
		}
	}

	var lng *float64
	if lngStr := query.Get("lng"); lngStr != "" {
		if val, err := strconv.ParseFloat(lngStr, 64); err == nil {
			lng = &val
		}
	}

	var radius *float64
	radStr := query.Get("radius_km")
	if radStr == "" {
		radStr = query.Get("radius")
	}
	if radStr != "" {
		if val, err := strconv.ParseFloat(radStr, 64); err == nil {
			radius = &val
		}
	}

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := dto.ShopFilter{
		Search:   search,
		Category: category,
		City:     city,
		Pincode:  pincode,
		Lat:      lat,
		Lng:      lng,
		RadiusKm: radius,
		Page:     page,
		Limit:    limit,
	}

	result, err := c.service.ListShops(r.Context(), filter)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shops retrieved successfully", result)
}

// GetByID handles retrieving a single shop by ID (Public)
func (c *ShopController) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		reuse.Error(w, http.StatusBadRequest, "shop id is required")
		return
	}

	shop, err := c.service.GetShopByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop retrieved successfully", shop)
}

// GetBySlug handles retrieving a single shop by its SEO slug (Public)
func (c *ShopController) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	shop, err := c.service.GetShopBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop retrieved successfully", shop)
}

// GetShopQR generates and serves a scannable PNG QR code for a shop by slug (Public)
func (c *ShopController) GetShopQR(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	pngBytes, err := c.service.GenerateShopQRCode(r.Context(), slug)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "shop not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s-qr.png"`, slug))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pngBytes)
}

// GetMyShopQR generates and serves a scannable PNG QR code for the authenticated shop owner (Protected)
func (c *ShopController) GetMyShopQR(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	pngBytes, err := c.service.GenerateMyShopQRCode(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `inline; filename="my-shop-qr.png"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pngBytes)
}

// GetDailyDigest returns today's aggregated retail overview (Protected - Shop)
func (c *ShopController) GetDailyDigest(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	digest, err := c.service.GetDailyDigest(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Daily retail digest retrieved successfully", digest)
}

// GetMerchantDashboard returns the consolidated dukandar overview in 1 single HTTP call (Protected - Shop)
func (c *ShopController) GetMerchantDashboard(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dashboard, err := c.service.GetMerchantDashboard(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, services.ErrShopNotFound) {
			reuse.Error(w, http.StatusNotFound, "you have not registered a shop yet")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Merchant dashboard data retrieved successfully", dashboard)
}

// GetHomeFeed returns the consolidated customer storefront feed in 1 single HTTP call (Public)
func (c *ShopController) GetHomeFeed(w http.ResponseWriter, r *http.Request) {
	var lat, lng *float64
	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		if val, err := strconv.ParseFloat(latStr, 64); err == nil {
			lat = &val
		}
	}
	if lngStr := r.URL.Query().Get("lng"); lngStr != "" {
		if val, err := strconv.ParseFloat(lngStr, 64); err == nil {
			lng = &val
		}
	}

	city := r.URL.Query().Get("city")
	limitShops := 6
	if limitStr := r.URL.Query().Get("limit_shops"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limitShops = val
		}
	}
	limitProducts := 20
	if limitStr := r.URL.Query().Get("limit_products"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limitProducts = val
		}
	}

	feed, err := c.service.GetHomeFeed(r.Context(), lat, lng, city, limitShops, limitProducts)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Customer home feed retrieved successfully", feed)
}

