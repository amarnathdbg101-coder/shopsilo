package repository

import (
	"shopMe/internal/handler/model"
	"testing"
	"time"

	sqlscan "github.com/amarnathdbg101-coder/go/go-sqlscan"
)

func TestSqlScan_CategoryHydration(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	parentID := "cat_root_001"
	shopID := "shop_999"

	columns := []string{
		"id", "name", "slug", "description", "image_url", "icon",
		"shop_id", "is_custom", "parent_id", "is_active", "created_at",
		"unrecognized_db_column", // Extra column from database should be safely ignored
	}

	mockValues := []any{
		"cat_123",
		"Dairy & Sweets",
		"dairy-sweets",
		"Fresh cow milk and authentic Indian sweets",
		"https://r2.shopsilo.in/dairy.jpg",
		"🥛",
		&shopID,
		true,
		&parentID,
		true,
		now,
		"some_raw_unmapped_blob",
	}

	var cat model.Category
	err := sqlscan.ScanWithColumns(columns, func(dest ...any) error {
		if len(dest) != len(mockValues) {
			t.Fatalf("dest count %d != value count %d", len(dest), len(mockValues))
		}
		for i := range dest {
			switch ptr := dest[i].(type) {
			case *string:
				*ptr = mockValues[i].(string)
			case **string:
				*ptr = mockValues[i].(*string)
			case *bool:
				*ptr = mockValues[i].(bool)
			case *time.Time:
				*ptr = mockValues[i].(time.Time)
			case *any:
				*ptr = mockValues[i]
			}
		}
		return nil
	}, &cat)

	if err != nil {
		t.Fatalf("ScanWithColumns failed: %v", err)
	}

	if cat.ID != "cat_123" {
		t.Errorf("ID mismatch: got %s, want cat_123", cat.ID)
	}
	if cat.Name != "Dairy & Sweets" {
		t.Errorf("Name mismatch: got %s, want Dairy & Sweets", cat.Name)
	}
	if cat.Slug != "dairy-sweets" {
		t.Errorf("Slug mismatch: got %s, want dairy-sweets", cat.Slug)
	}
	if cat.Icon != "🥛" {
		t.Errorf("Icon mismatch: got %s, want 🥛", cat.Icon)
	}
	if cat.ParentID == nil || *cat.ParentID != parentID {
		t.Errorf("ParentID pointer mismatch: got %v, want %s", cat.ParentID, parentID)
	}
	if !cat.IsActive {
		t.Errorf("IsActive mismatch: got false, want true")
	}
	if !cat.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", cat.CreatedAt, now)
	}
}

func TestSqlScan_ShopExpenseHydration(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	columns := []string{
		"id", "shop_id", "category", "amount", "notes",
		"payment_method", "expense_date", "created_at",
	}

	mockValues := []any{
		"exp_555",
		"shp_101",
		"electricity",
		float64(3450.50),
		"October shop power bill",
		"upi",
		"2026-11-08",
		now,
	}

	var exp model.ShopExpense
	err := sqlscan.ScanWithColumns(columns, func(dest ...any) error {
		for i := range dest {
			switch ptr := dest[i].(type) {
			case *string:
				*ptr = mockValues[i].(string)
			case *float64:
				*ptr = mockValues[i].(float64)
			case *time.Time:
				*ptr = mockValues[i].(time.Time)
			case *any:
				*ptr = mockValues[i]
			}
		}
		return nil
	}, &exp)

	if err != nil {
		t.Fatalf("ScanWithColumns failed: %v", err)
	}

	if exp.ID != "exp_555" || exp.ShopID != "shp_101" || exp.Category != "electricity" {
		t.Errorf("expense core fields mismatch: %+v", exp)
	}
	if exp.Amount != 3450.50 {
		t.Errorf("amount mismatch: got %f, want 3450.50", exp.Amount)
	}
	if exp.PaymentMethod != "upi" || exp.ExpenseDate != "2026-11-08" {
		t.Errorf("expense date/method mismatch: %+v", exp)
	}
}
