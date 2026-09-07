package utils_test

import (
	"bytes"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"testing"
)

func TestGenerateWholesaleReorderPDF(t *testing.T) {
	shop := &model.Shop{
		Name:           "Sharma Electronics & General Store",
		Address:        "Shop No. 4, Main Market",
		City:           "Delhi",
		Pincode:        "110001",
		Phone:          "+91 9876543210",
		WhatsAppNumber: "+91 9876543210",
	}

	items := []*dto.LowStockProduct{
		{
			ProductID:           "b9c02052-a5e2-45a7-96a9-e85d45db60ea",
			Name:                "Wireless Bluetooth Earbuds",
			SKU:                 "EAR-001",
			CurrentStock:        2,
			ReservedStock:       1,
			AvailableStock:      1,
			LowStockThreshold:   10,
			SuggestedReorderQty: 25,
			Price:               999.0,
		},
		{
			ProductID:           "c1f7b049-382a-4a21-8bc1-192837465012",
			Name:                "Fast Charging USB-C Cable",
			SKU:                 "CAB-002",
			CurrentStock:        0,
			ReservedStock:       0,
			AvailableStock:      0,
			LowStockThreshold:   15,
			SuggestedReorderQty: 50,
			Price:               249.0,
		},
	}

	pdfBytes, err := utils.GenerateWholesaleReorderPDF(shop, items)
	if err != nil {
		t.Fatalf("unexpected error generating PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty pdf bytes")
	}

	// Verify standard PDF header: starts with %PDF-
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("generated file is not a valid PDF")
	}
}
