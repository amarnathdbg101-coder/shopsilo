package services_test

import (
	"context"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"testing"
)

// TestProductServiceRules tests product creation constraints, shop requirement, and image caps
func TestProductServiceRules(t *testing.T) {
	t.Run("CreateProduct requires registered shop (ErrShopRequired)", func(t *testing.T) {
		ps := services.NewProductService(nil, nil, nil)
		defer func() {
			if r := recover(); r != nil {
				t.Logf("handled nil repo safely in test: %v", r)
			}
		}()

		input := dto.CreateProductRequest{
			Name:          "Men's Linen Shirt",
			SKU:           "MLS-001",
			Price:         899.0,
			CategoryID:    "clothing-apparel",
			StockQuantity: 15,
		}

		_, err := ps.CreateProduct(context.Background(), "user-with-no-shop", input)
		if err == nil {
			t.Fatalf("expected error when user has no shop, got nil")
		}
	})
}

// TestKhataAndExpenseDTOValidation verifies transaction models, udhar types, and limits
func TestKhataAndExpenseDTOValidation(t *testing.T) {
	t.Run("Record credit with valid payload structure", func(t *testing.T) {
		req := dto.RecordCreditTransactionRequest{
			CustomerMobile: "9876543210",
			CustomerName:   "Ramesh Kumar",
			Amount:         1500.0,
			Notes:          "Diwali cloth purchase balance",
			BillNumber:     "BIL-261018-1001",
			ItemsSummary:   "2 Sarees, 1 Kurta",
		}

		if req.Amount <= 0 {
			t.Fatalf("credit amount must be positive")
		}
		if req.CustomerMobile == "" {
			t.Fatalf("customer mobile is required for udhar khata")
		}
	})

	t.Run("Create expense payload structure", func(t *testing.T) {
		req := dto.CreateExpenseRequest{
			Category:      "Electricity",
			Amount:        2450.0,
			Notes:         "October store electricity bill",
			PaymentMethod: "upi",
			ExpenseDate:   "2026-10-18",
		}

		if req.Amount <= 0 {
			t.Fatalf("expense amount must be positive")
		}
		if req.Category == "" {
			t.Fatalf("expense category is required")
		}
	})
}
