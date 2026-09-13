package services

import (
	"testing"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

func TestValidatePOSItemsRejectsEmptyProductIDsAndZeroQuantity(t *testing.T) {
	items := []dto.POSSaleItemRequest{{ProductID: "", Quantity: 1}, {ProductID: "prod-1", Quantity: 0}}

	if err := validatePOSItems(items); err == nil {
		t.Fatal("expected validation error for invalid POS items")
	}
}

func TestCalculateParkedCartTotalUsesCustomPriceAndBatchProductValues(t *testing.T) {
	items := []dto.POSSaleItemRequest{
		{ProductID: "p1", Quantity: 2},
		{ProductID: "p2", Quantity: 1, CustomPrice: floatPtr(125.5)},
	}

	products := map[string]*model.Product{
		"p1": {ID: "p1", Price: 80},
		"p2": {ID: "p2", Price: 200},
	}

	total, err := calculateParkedCartTotal(items, products)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 285.5 {
		t.Fatalf("expected total 285.5, got %.2f", total)
	}
}

func TestNormalizePOSItemsRejectsNegativeCustomPrice(t *testing.T) {
	items := []dto.POSSaleItemRequest{{ProductID: "p1", Quantity: 2, CustomPrice: floatPtr(-1)}}

	if _, err := normalizePOSItems(items); err == nil {
		t.Fatal("expected negative custom price to be rejected")
	}
}

func floatPtr(v float64) *float64 { return &v }
