package utils_test

import (
	"testing"

	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
)

func TestGenerateStoreStandeePDF(t *testing.T) {
	t.Run("Standard Store Standee", func(t *testing.T) {
		shop := &model.Shop{
			ID:             "shop-standee-123",
			Name:           "Om Clothes & Cosmetics",
			Slug:           "om-clothes",
			Category:       "Apparel & Cosmetics",
			City:           "Darbhanga",
			Address:        "Dhoi, Pipal Chowk, Darbhanga, Bihar",
			Phone:          "9876543210",
			WhatsAppNumber: "9876543210",
			UPIID:          "omclothes@okhdfcbank",
		}

		pdfBytes, err := utils.GenerateStoreStandeePDF(shop)
		if err != nil {
			t.Fatalf("GenerateStoreStandeePDF failed: %v", err)
		}
		if len(pdfBytes) == 0 {
			t.Fatalf("expected non-empty PDF bytes")
		}
		if string(pdfBytes[:4]) != "%PDF" {
			t.Errorf("expected valid PDF header %%%%PDF, got %s", string(pdfBytes[:4]))
		}
		t.Logf("Successfully generated store standee PDF: %d bytes", len(pdfBytes))
	})

	t.Run("Unicode and Emojis Resilience", func(t *testing.T) {
		shop := &model.Shop{
			Name:           "Om Clothes & Cosmetics 🏪👗",
			Slug:           "om-clothes",
			Category:       "Clothing & Beauty",
			Address:        "Near City Mall — Shop #12 “Best Quality”",
			Phone:          "9876543210",
			WhatsAppNumber: "9876543210",
			UPIID:          "omstore@okhdfcbank",
		}
		pdfBytes, err := utils.GenerateStoreStandeePDF(shop)
		if err != nil {
			t.Fatalf("failed with unicode: %v", err)
		}
		if len(pdfBytes) == 0 || string(pdfBytes[:4]) != "%PDF" {
			t.Errorf("invalid PDF output")
		}
	})

	t.Run("Minimal Store Without Optional Fields", func(t *testing.T) {
		shop := &model.Shop{
			Name: "Quick Mart",
			Slug: "quick-mart",
		}
		pdfBytes, err := utils.GenerateStoreStandeePDF(shop)
		if err != nil {
			t.Fatalf("failed with minimal fields: %v", err)
		}
		if len(pdfBytes) == 0 || string(pdfBytes[:4]) != "%PDF" {
			t.Errorf("invalid PDF output")
		}
	})
}
