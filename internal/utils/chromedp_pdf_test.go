package utils_test

import (
	"bytes"
	"context"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"testing"
	"time"
)

func TestBuildInvoiceHTML(t *testing.T) {
	bill := &model.POSBill{
		BillNumber:     "BIL-TEST-0001",
		CustomerPhone:  "9876543210",
		PaymentMethod:  "split",
		Subtotal:       500.0,
		DiscountAmount: 50.0,
		TotalAmount:    450.0,
		CashAmount:     200.0,
		OnlineAmount:   150.0,
		KhataAmount:    100.0,
		CreatedAt:      time.Now(),
		Shop: &model.Shop{
			Name:            "Sharma General Store",
			Phone:           "9876543210",
			Address:         "Shop #4, Market Complex",
			City:            "Jaipur",
			Pincode:         "302001",
			WhatsAppNumber: "9876543210",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Tata Salt 1kg (टाटा नमक)",
				ProductSKU:  "TATASALT01",
				Quantity:    2,
				UnitPrice:   28.0,
				TotalPrice:  56.0,
			},
			{
				ProductName: "Fortune Sunlite Oil 1L (फॉर्च्यून तेल)",
				ProductSKU:  "FORT01",
				Quantity:    3,
				UnitPrice:   135.0,
				TotalPrice:  405.0,
			},
		},
	}

	html := utils.BuildInvoiceHTML(bill)
	if html == "" {
		t.Fatalf("expected non-empty HTML invoice template")
	}

	if !bytes.Contains([]byte(html), []byte("Sharma General Store")) {
		t.Errorf("HTML missing store name")
	}
	if !bytes.Contains([]byte(html), []byte("BIL-TEST-0001")) {
		t.Errorf("HTML missing bill number")
	}
	if !bytes.Contains([]byte(html), []byte("Tata Salt")) {
		t.Errorf("HTML missing item name")
	}
}

func TestGenerateSmartBillPDF(t *testing.T) {
	bill := &model.POSBill{
		BillNumber:     "BIL-SMART-9999",
		CustomerPhone:  "9988776655",
		PaymentMethod:  "cash",
		Subtotal:       120.0,
		DiscountAmount: 0.0,
		TotalAmount:    120.0,
		CashAmount:     120.0,
		CreatedAt:      time.Now(),
		Shop: &model.Shop{
			Name:    "Apex Store",
			Address: "Station Road",
			City:    "Delhi",
			Pincode: "110001",
			Phone:   "9988776655",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Aashirvaad Atta 5kg",
				Quantity:    1,
				UnitPrice:   120.0,
				TotalPrice:  120.0,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pdfBytes, err := utils.GenerateSmartBillPDF(ctx, bill)
	if err != nil {
		t.Fatalf("GenerateSmartBillPDF failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty PDF bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("invalid PDF magic header bytes: got %v", pdfBytes[:10])
	}
}
