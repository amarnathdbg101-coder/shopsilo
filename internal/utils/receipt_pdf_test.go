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

func TestGenerateKhataStatementPDF(t *testing.T) {
	customer := &model.CustomerKhata{
		CustomerName:   "Aakash Verma",
		CustomerMobile: "9876543210",
		CurrentBalance: 1450.0,
		CreditLimit:    5000.0,
		UpdatedAt:      time.Now(),
	}

	shop := &model.Shop{
		Name:           "Verma General Store",
		Address:        "Shop #4, Main Bazaar, Lucknow",
		Phone:          "9876500000",
		WhatsAppNumber: "9876500000",
	}

	transactions := []model.KhataTransaction{
		{
			Type:         model.KhataTxTypeGiveCredit,
			Amount:       1000.0,
			BalanceAfter: 1000.0,
			Notes:        "Groceries monthly supply",
			CreatedAt:    time.Now().Add(-48 * time.Hour),
		},
		{
			Type:         model.KhataTxTypeGiveCredit,
			Amount:       750.0,
			BalanceAfter: 1750.0,
			BillNumber:   "BIL-260909-1234",
			CreatedAt:    time.Now().Add(-24 * time.Hour),
		},
		{
			Type:         model.KhataTxTypeReceivePayment,
			Amount:       300.0,
			BalanceAfter: 1450.0,
			PaymentMode:  "upi",
			CreatedAt:    time.Now(),
		},
	}

	pdfBytes, err := utils.GenerateKhataStatementPDF(customer, transactions, shop)
	if err != nil {
		t.Fatalf("unexpected error generating statement PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty pdf bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("generated statement is not a valid PDF")
	}
}

func TestGeneratePOSReceiptPDF_SplitPayment(t *testing.T) {
	bill := &model.POSBill{
		BillNumber:     "POS-260909-SPLIT",
		CustomerPhone:  "+91 9876543210",
		Subtotal:       1500.0,
		DiscountAmount: 100.0,
		TotalAmount:    1400.0,
		PaymentMethod:  "split",
		CashAmount:     500.0,
		OnlineAmount:   500.0,
		KhataAmount:    400.0,
		CreatedAt:      time.Now(),
		Shop: &model.Shop{
			Name:           "Sharma Kirana & General Store",
			Address:        "Sector 18, Noida",
			Phone:          "9876543210",
			WhatsAppNumber: "9876543210",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Basmati Rice 5kg",
				Quantity:    1,
				UnitPrice:   600.0,
				TotalPrice:  600.0,
			},
			{
				ProductName: "Fortune Mustard Oil 5L",
				Quantity:    1,
				UnitPrice:   800.0,
				TotalPrice:  800.0,
			},
		},
	}

	pdfBytes, err := utils.GeneratePOSReceiptPDF(bill)
	if err != nil {
		t.Fatalf("unexpected error generating split receipt PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty pdf bytes")
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("generated receipt is not a valid PDF")
	}
}

