// Package model handle database struct.
package model

import "time"

type Shop struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description,omitempty"`
	Category       string    `json:"category,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	Address        string    `json:"address,omitempty"`
	Latitude       *float64  `json:"latitude,omitempty"`
	Longitude      *float64  `json:"longitude,omitempty"`
	City           string    `json:"city,omitempty"`
	Pincode        string    `json:"pincode,omitempty"`
	WhatsAppNumber string    `json:"whatsapp_number,omitempty"`
	LogoURL        string    `json:"logo_url,omitempty"`
	Banners        []string  `json:"banners"` // max 2 promotional banners
	Timing         string    `json:"timing,omitempty"`
	OpeningTime    string    `json:"opening_time,omitempty"`
	ClosingTime    string    `json:"closing_time,omitempty"`
	WeeklyOff      string    `json:"weekly_off,omitempty"`
	IsOpen         bool      `json:"is_open"`
	IsCurrentlyOpen bool     `json:"is_currently_open"`
	IsActive       bool      `json:"is_active"`
	DistanceKm     *float64  `json:"distance_km,omitempty"`
	GoogleMapsURL  string    `json:"google_maps_url,omitempty"`
	WhatsAppURL    string    `json:"whatsapp_url,omitempty"`
	AverageRating  float64   `json:"average_rating"`
	TotalReviews   int       `json:"total_reviews"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
