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
	UPIID          string    `json:"upi_id,omitempty"`
	LogoURL        string    `json:"logo_url,omitempty"`
	Banners        []string  `json:"banners"` // max 4 promotional banners
	Timing         string    `json:"timing,omitempty"`
	OpeningTime    string    `json:"opening_time,omitempty"`
	ClosingTime    string    `json:"closing_time,omitempty"`
	WeeklyOff      string    `json:"weekly_off,omitempty"`
	IsOpen          bool      `json:"is_open"`
	IsCurrentlyOpen bool      `json:"is_currently_open"`
	LiveStatus      string    `json:"live_status,omitempty"`     // OPEN, CLOSING_SOON, CLOSED, WEEKLY_OFF
	StatusMessage   string    `json:"status_message,omitempty"`  // "Open now • Closes at 9:00 PM"
	NextChangeAt    string    `json:"next_change_at,omitempty"`  // "Closes in 22 mins" or "Opens tomorrow at 9:00 AM"
	MinutesToClose  int       `json:"minutes_to_close,omitempty"`// e.g. 22
	IsActive        bool      `json:"is_active"`
	DistanceKm      *float64  `json:"distance_km,omitempty"`
	GoogleMapsURL   string    `json:"google_maps_url,omitempty"`
	WhatsAppURL     string    `json:"whatsapp_url,omitempty"`
	UPIPaymentURL   string    `json:"upi_payment_url,omitempty"` // Standard NPCI UPI Deep Link (upi://pay?...)
	AverageRating  float64   `json:"average_rating"`
	TotalReviews   int       `json:"total_reviews"`
	Status            string    `json:"status"` // active, pending_review, flagged, suspended, banned
	FlaggedCount      int       `json:"flagged_count"`
	SuspensionReason  string    `json:"suspension_reason,omitempty"`
	CreationIP        string    `json:"-"`
	CreationUserAgent string    `json:"-"`
	DeviceFingerprint string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ShopDailyDigest struct {
	Date                string  `json:"date"`
	TodaySalesAmount    float64 `json:"today_sales_amount"`
	TodaySalesCount     int     `json:"today_sales_count"`
	ActiveReservations  int     `json:"active_reservations"`
	LowStockCount       int     `json:"low_stock_count"`
	TotalKhataUdhar     float64 `json:"total_khata_udhar"`
	TotalKhataCustomers int     `json:"total_khata_customers"`
}
