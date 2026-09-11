// Package dto handler request and response struct.
package dto

import "shopMe/internal/handler/model"

type CreateShopRequest struct {
	Name           string    `json:"name" validate:"required,min=2,max=150"`
	Slug           string    `json:"slug,omitempty" validate:"omitempty,min=2,max=160"`
	Description    string    `json:"description,omitempty" validate:"omitempty,max=500"`
	Category       string    `json:"category" validate:"required,min=2,max=100"`
	Phone          string    `json:"phone,omitempty" validate:"omitempty,max=20"`
	WhatsAppNumber string    `json:"whatsapp_number,omitempty" validate:"omitempty,max=20"`
	Address        string    `json:"address" validate:"required,min=5,max=255"`
	City           string    `json:"city,omitempty" validate:"omitempty,max=100"`
	Pincode        string    `json:"pincode,omitempty" validate:"omitempty,max=20"`
	Latitude       *float64  `json:"latitude,omitempty" validate:"omitempty,latitude"`
	Longitude      *float64  `json:"longitude,omitempty" validate:"omitempty,longitude"`
	LogoURL        string    `json:"logo_url,omitempty"`
	Banners        []string  `json:"banners,omitempty" validate:"omitempty,max=2"` // max 2 promotional banners
	Timing         string    `json:"timing,omitempty" validate:"omitempty,max=100"`
	OpeningTime    string    `json:"opening_time,omitempty" validate:"omitempty,max=10"`
	ClosingTime    string    `json:"closing_time,omitempty" validate:"omitempty,max=10"`
	WeeklyOff      string    `json:"weekly_off,omitempty" validate:"omitempty,max=20"`
}

type UpdateShopRequest struct {
	Name           *string   `json:"name,omitempty" validate:"omitempty,min=2,max=150"`
	Slug           *string   `json:"slug,omitempty" validate:"omitempty,min=2,max=160"`
	Description    *string   `json:"description,omitempty" validate:"omitempty,max=500"`
	Category       *string   `json:"category,omitempty" validate:"omitempty,min=2,max=100"`
	Phone          *string   `json:"phone,omitempty" validate:"omitempty,max=20"`
	WhatsAppNumber *string   `json:"whatsapp_number,omitempty" validate:"omitempty,max=20"`
	Address        *string   `json:"address,omitempty" validate:"omitempty,min=5,max=255"`
	City           *string   `json:"city,omitempty" validate:"omitempty,max=100"`
	Pincode        *string   `json:"pincode,omitempty" validate:"omitempty,max=20"`
	Latitude       *float64  `json:"latitude,omitempty" validate:"omitempty,latitude"`
	Longitude      *float64  `json:"longitude,omitempty" validate:"omitempty,longitude"`
	LogoURL        *string   `json:"logo_url,omitempty"`
	Banners        *[]string `json:"banners,omitempty" validate:"omitempty,max=2"` // max 2 promotional banners
	Timing         *string   `json:"timing,omitempty" validate:"omitempty,max=100"`
	OpeningTime    *string   `json:"opening_time,omitempty" validate:"omitempty,max=10"`
	ClosingTime    *string   `json:"closing_time,omitempty" validate:"omitempty,max=10"`
	WeeklyOff      *string   `json:"weekly_off,omitempty" validate:"omitempty,max=20"`
	IsOpen         *bool     `json:"is_open,omitempty"`
}

type ToggleShopStatusRequest struct {
	IsOpen bool `json:"is_open"`
}

type ShopFilter struct {
	Search   string   `json:"search,omitempty"`
	Category string   `json:"category,omitempty"`
	City     string   `json:"city,omitempty"`
	Pincode  string   `json:"pincode,omitempty"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	RadiusKm *float64 `json:"radius_km,omitempty"`
	Page     int      `json:"page,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type ShopPaginationResponse struct {
	Shops      []*model.Shop `json:"shops"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

type ShopResponse struct {
	Shop        *model.Shop `json:"shop"`
	AccessToken string      `json:"access_token,omitempty"`
}

type MerchantDashboardResponse struct {
	Shop              *model.Shop              `json:"shop"`
	Digest            *model.ShopDailyDigest   `json:"digest"`
	WeeklyScorecard   *WeeklyScorecardResponse `json:"weekly_scorecard,omitempty"`
	ActiveOffersCount int                      `json:"active_offers_count"`
}

type HomeFeedResponse struct {
	Categories []*model.Category `json:"categories"`
	Shops      []*model.Shop     `json:"shops"`
	Products   []*model.Product  `json:"products"`
}
