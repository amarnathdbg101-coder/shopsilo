package utils_test

import (
	"bytes"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"testing"
	"time"
)

func TestGeneratePOSReceiptPDF(t *testing.T) {
	bill := &model.POSBill{
		BillNumber:     "POS-260907-0042",
		CustomerPhone:  "+91 9876543210",
		Subtotal:       1200.0,
		DiscountAmount: 50.0,
		TotalAmount:    1150.0,
		PaymentMethod:  "upi",
		CreatedAt:      time.Now(),
		Shop: &model.Shop{
			Name:           "Delhi Super Market",
			Address:        "12 Connaught Place, New Delhi",
			Phone:          "+91 1123456789",
			WhatsAppNumber: "+91 9876543210",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Organic Green Tea 250g",
				Quantity:    2,
				UnitPrice:   350.0,
				TotalPrice:  700.0,
			},
			{
				ProductName: "Almond Cookies 500g",
				Quantity:    1,
				UnitPrice:   500.0,
				TotalPrice:  500.0,
			},
		},
	}

	pdfBytes, err := utils.GeneratePOSReceiptPDF(bill)
	if err != nil {
		t.Fatalf("unexpected error generating receipt PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty pdf bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("generated receipt is not a valid PDF")
	}
}
