package utils_test

import (
	"bytes"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
)

func samplePOSBill() *model.POSBill {
	return &model.POSBill{
		BillNumber:     "BIL-NATIVE-8888",
		CustomerPhone:  "9876543210",
		PaymentMethod:  "split",
		Subtotal:       450.0,
		DiscountAmount: 30.0,
		TotalAmount:    420.0,
		CashAmount:     200.0,
		OnlineAmount:   120.0,
		KhataAmount:    100.0,
		CreatedAt:      time.Now(),
		Shop: &model.Shop{
			Name:           "Sharma Ji General Store (शर्मा जी किराना)",
			Address:        "Shop #4, Gandhi Chowk",
			City:           "Kanpur",
			Pincode:        "208001",
			Phone:          "9876543210",
			WhatsAppNumber: "9876543210",
			UPIID:          "sharma@upi",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Aashirvaad Shudh Chakki Atta 5kg",
				ProductSKU:  "ATTA-01",
				Quantity:    1,
				UnitPrice:   240.0,
				TotalPrice:  240.0,
			},
			{
				ProductName: "Fortune Sunlite Refined Oil 1L",
				ProductSKU:  "OIL-SUN-02",
				Quantity:    1,
				UnitPrice:   150.0,
				TotalPrice:  150.0,
			},
			{
				ProductName: "Tata Salt 1kg (टाटा नमक)",
				ProductSKU:  "SALT-TATA",
				Quantity:    2,
				UnitPrice:   30.0,
				TotalPrice:  60.0,
			},
		},
	}
}

func TestGenerateNativeBillPDF(t *testing.T) {
	bill := samplePOSBill()

	start := time.Now()
	pdfBytes, err := utils.GenerateNativeBillPDF(bill, utils.NativePDFOptions{Format: utils.FormatA5})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeBillPDF failed: %v", err)
	}

	if len(pdfBytes) < 500 {
		t.Fatalf("Generated PDF is suspiciously small: %d bytes", len(pdfBytes))
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Generated file does not start with %%PDF- magic header")
	}

	t.Logf("✅ Native A5 Bill PDF generated successfully in %v (%d KB)", duration, len(pdfBytes)/1024)
}

func TestGenerateNativeThermalBillPDF(t *testing.T) {
	bill := samplePOSBill()

	start := time.Now()
	pdfBytes, err := utils.GenerateNativeThermalBillPDF(bill)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeThermalBillPDF failed: %v", err)
	}

	if len(pdfBytes) < 300 {
		t.Fatalf("Thermal PDF too small: %d bytes", len(pdfBytes))
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Thermal PDF does not start with %%PDF- header")
	}

	t.Logf("✅ Native 80mm Thermal Receipt generated successfully in %v (%d bytes)", duration, len(pdfBytes))
}

func TestGenerateNativeKhataStatementPDF(t *testing.T) {
	shop := samplePOSBill().Shop
	customer := &model.CustomerKhata{
		CustomerName:   "Ramesh Kumar (रमेश कुमार)",
		CustomerMobile: "9876543210",
		CurrentBalance: 1250.0,
	}

	transactions := []model.KhataTransaction{
		{
			CreatedAt:  time.Now().Add(-48 * time.Hour),
			Type:       "GIVE_CREDIT",
			Amount:     800.0,
			Notes:      "Rashan & Oil",
			BillNumber: "BIL-101",
		},
		{
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			Type:       "GIVE_CREDIT",
			Amount:     750.0,
			Notes:      "Wheat & Spices",
			BillNumber: "BIL-105",
		},
		{
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Type:      "RECEIVE_PAYMENT",
			Amount:    300.0,
			Notes:     "Paid via PhonePe",
		},
	}

	start := time.Now()
	pdfBytes, err := utils.GenerateNativeKhataStatementPDF(customer, transactions, shop)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeKhataStatementPDF failed: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Khata PDF missing %%PDF- prefix")
	}

	t.Logf("✅ Native Khata Statement PDF generated successfully in %v (%d KB)", duration, len(pdfBytes)/1024)
}

func TestGenerateNativeWholesaleReorderPDF(t *testing.T) {
	shop := samplePOSBill().Shop
	items := []*dto.LowStockProduct{
		{
			Name:                "Aashirvaad Shudh Chakki Atta 10kg",
			SKU:                 "ATTA-10",
			CurrentStock:        2,
			SuggestedReorderQty: 25,
		},
		{
			Name:                "Fortune Refined Soyabean Oil 1L",
			SKU:                 "OIL-SOYA",
			CurrentStock:        4,
			SuggestedReorderQty: 50,
		},
		{
			Name:                "Sugar / Cheeni 50kg Bora (Loose)",
			SKU:                 "SUG-50",
			CurrentStock:        1,
			SuggestedReorderQty: 5,
		},
	}

	start := time.Now()
	pdfBytes, err := utils.GenerateNativeWholesaleReorderPDF(shop, items)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeWholesaleReorderPDF failed: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Wholesale PDF missing %%PDF- prefix")
	}

	t.Logf("✅ Native Wholesale Reorder PDF generated successfully in %v (%d KB)", duration, len(pdfBytes)/1024)
}

func TestGenerateNativeStoreStandeePDF(t *testing.T) {
	shop := samplePOSBill().Shop

	start := time.Now()
	pdfBytes, err := utils.GenerateNativeStoreStandeePDF(shop)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeStoreStandeePDF failed: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Standee PDF missing %%PDF- prefix")
	}

	t.Logf("✅ Native Store Standee PDF generated successfully in %v (%d KB)", duration, len(pdfBytes)/1024)
}

func TestGenerateNativeCustomProcurementPDF_MultilingualAndBlankResilience(t *testing.T) {
	// Exact items representing user's actual problematic dataset
	items := []dto.ProcurementPDFItem{
		{Name: "Item 1", Qty: "10", EstimatedPrice: "₹20"},
		{Name: "", Category: " ", Qty: "1", EstimatedPrice: "₹10"},
		{Name: "", Category: "", Qty: "24", EstimatedPrice: "₹10"},
		{Name: "", Qty: "24", EstimatedPrice: "₹10"},
		{Name: "", Qty: "1", EstimatedPrice: "₹10"},
		{Name: "", Qty: "24", EstimatedPrice: "₹10"},
		{Name: "1", Category: "", Qty: "12", EstimatedPrice: "₹0"},
		{Name: "500", Category: "", Qty: "1 2", EstimatedPrice: "₹0"},
		{Name: "किसान हुक", Category: "सामान्य किराना", Qty: "1", EstimatedPrice: "₹20"},
		{Name: "उन", Category: "कपड़ा और सिलाई", Qty: "10", EstimatedPrice: "₹130"},
		{Name: "बैंगनी ऊन", Category: "ऊन", Qty: "20", EstimatedPrice: "₹20"},
		{Name: "-", Category: "", Qty: "10", EstimatedPrice: "₹10"},
		{Name: "", Qty: "1", EstimatedPrice: "₹0"},
	}

	start := time.Now()
	// Test with shop == nil and title containing shop name
	pdfBytes, err := utils.GenerateNativeCustomProcurementPDF(nil, "Om Clothes and Cosmetic - Mandi Procurement Sheet", items)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNativeCustomProcurementPDF failed: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Procurement PDF missing %%PDF- prefix")
	}

	t.Logf("✅ Native Mandi Procurement PDF (13 complex items) generated in %v (%d KB)", duration, len(pdfBytes)/1024)
}
