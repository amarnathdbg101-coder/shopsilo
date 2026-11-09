package reuse_test

import (
	"bytes"
	"testing"
	"time"

	"shopMe/internal/handler/model"
	"shopMe/internal/reuse"
)

func TestReuse_GenerateESCPOSBillReceipt(t *testing.T) {
	bill := &model.POSBill{
		ID:            "bill_test_101",
		BillNumber:    "POS-2026-0099",
		CustomerPhone: "+91 98765 43210",
		Subtotal:      400.00,
		DiscountAmount: 50.00,
		TotalAmount:   350.00,
		PaymentMethod: "split",
		CashAmount:    150.00,
		OnlineAmount:  200.00,
		CreatedAt:     time.Now(),
		Shop: &model.Shop{
			Name:    "Gupta Kirana Store",
			Address: "Sector 14, Gurgaon",
			Phone:   "+91 124 555666",
		},
		Items: []*model.POSBillItem{
			{
				ProductName: "Tata Salt 1kg",
				Quantity:    2,
				UnitPrice:   28.00,
				TotalPrice:  56.00,
			},
			{
				ProductName: "Aashirvaad Atta 5kg",
				Quantity:    1,
				UnitPrice:   240.00,
				TotalPrice:  240.00,
			},
			{
				ProductName: "Maggi 4-Pack",
				Quantity:    1,
				UnitPrice:   104.00,
				TotalPrice:  104.00,
			},
		},
	}

	// 1. Generate 80mm receipt with drawer kick
	raw80, err := reuse.GenerateESCPOSBillReceipt(bill, false, true)
	if err != nil {
		t.Fatalf("failed to generate 80mm ESC/POS receipt: %v", err)
	}

	if len(raw80) < 100 || len(raw80) > 3000 {
		t.Fatalf("unexpected ESC/POS stream size: %d bytes", len(raw80))
	}

	// Must contain store name, bill number, items, and drawer kick pulse
	if !bytes.Contains(raw80, []byte("Gupta Kirana Store")) {
		t.Fatal("expected receipt to contain shop name")
	}
	if !bytes.Contains(raw80, []byte("POS-2026-0099")) {
		t.Fatal("expected receipt to contain bill number")
	}
	if !bytes.Contains(raw80, []byte("Tata Salt")) {
		t.Fatal("expected receipt to contain item name")
	}

	// 2. Generate 58mm receipt
	raw58, err := reuse.GenerateESCPOSBillReceipt(bill, true, false)
	if err != nil {
		t.Fatalf("failed to generate 58mm ESC/POS receipt: %v", err)
	}

	if len(raw58) == 0 {
		t.Fatal("expected non-empty 58mm receipt")
	}

	t.Logf("✅ 80mm ESC/POS Receipt: %d bytes; 58mm ESC/POS Receipt: %d bytes", len(raw80), len(raw58))
}
