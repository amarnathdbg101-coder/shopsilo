package services

import (
	"strings"
	"testing"

	"shopMe/internal/handler/model"
)

func TestEnrichShop(t *testing.T) {
	lat := 28.58285
	lng := 77.31103

	shop := &model.Shop{
		ID:             "test-shop-123",
		Name:           "Om Clothes & Cosmetics",
		Slug:           "om-clothes",
		Phone:          "9876543210",
		Address:        "Dhoi, Pipal Chowk, Darbhanga",
		Latitude:       &lat,
		Longitude:      &lng,
		OpeningTime:    "09:00",
		ClosingTime:    "21:00",
		WeeklyOff:      "Sunday",
		IsOpen:         true,
		UPIID:          "omclothes@okhdfcbank",
		WhatsAppNumber: "9876543210",
	}

	enrichShop(shop)

	// 1. Verify Real-time Timing Enrichment
	if shop.LiveStatus == "" {
		t.Errorf("expected LiveStatus to be populated, got empty string")
	}
	if shop.StatusMessage == "" {
		t.Errorf("expected StatusMessage to be populated, got empty string")
	}
	t.Logf("Shop Timing -> IsCurrentlyOpen: %t | LiveStatus: %s | Message: %s | NextChange: %s",
		shop.IsCurrentlyOpen, shop.LiveStatus, shop.StatusMessage, shop.NextChangeAt)

	// 2. Verify Google Maps URL
	expectedMapsURL := "https://www.google.com/maps/search/?api=1&query=28.582850,77.311030"
	if shop.GoogleMapsURL != expectedMapsURL {
		t.Errorf("expected GoogleMapsURL %s, got %s", expectedMapsURL, shop.GoogleMapsURL)
	}

	// 3. Verify Standard NPCI UPI Deep Link
	if !strings.HasPrefix(shop.UPIPaymentURL, "upi://pay?pa=omclothes@okhdfcbank") {
		t.Errorf("expected UPIPaymentURL to start with upi://pay, got %s", shop.UPIPaymentURL)
	}
	if !strings.Contains(shop.UPIPaymentURL, "mc=5411") {
		t.Errorf("expected UPIPaymentURL to contain mc=5411 merchant category code, got %s", shop.UPIPaymentURL)
	}
	t.Logf("Generated NPCI UPI URL: %s", shop.UPIPaymentURL)

	// 4. Verify WhatsApp URL with E.164 and prefilled greeting
	if !strings.HasPrefix(shop.WhatsAppURL, "https://wa.me/919876543210") {
		t.Errorf("expected WhatsAppURL to have +91 prefix, got %s", shop.WhatsAppURL)
	}
	if !strings.Contains(shop.WhatsAppURL, "shopsilo.in%2Fshop%2Fom-clothes") {
		t.Errorf("expected WhatsAppURL to contain encoded shop link, got %s", shop.WhatsAppURL)
	}
	t.Logf("Generated WhatsApp URL: %s", shop.WhatsAppURL)
}

