package services

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

func TestParseRecordsToImportItems_UserCSV(t *testing.T) {
	csvData := `Product Name,Category,Brand / Company,Selling Price (₹),Cost Price (Khared) (₹),MRP / Compare Price (₹),Discount % OFF MRP,Est. Profit (₹),Margin %,Stock Quantity,Unit,Low Stock Alert,Weight (Kg),Barcode / SKU,Attribute 1 Name,Attribute 1 Value,Attribute 2 Name,Attribute 2 Value,Description / Notes,Allow Customer Bargain,Featured Product Badge
VIM Dish Bar Lemon (Pack of 3),Household & Cleaning,VIM,42,34,45,6.7,8,19.0,25,Piece,5,0.6,VIMBAR03,Weight,200g x 3,Fragrance,Lemon,VIM dish bar with lemon power cuts tough grease fast and keeps utensils sparkling clean.,No,No
Surf Excel Easy Wash Detergent Powder 1kg,Household & Cleaning,Surf Excel,115,98,120,4.2,17,14.8,15,Piece,5,1.0,SURFEX01,Weight,1kg,Type,Detergent Powder,"Surf Excel Easy Wash removes tough stains in machines and by hand, suitable for all fabrics.",No,Yes
Harpic Original Toilet Cleaner 500ml,Household & Cleaning,Harpic,95,78,99,4.0,17,17.9,30,Piece,6,0.55,HARPIC50,Weight,500ml,Fragrance,Original,"Harpic toilet cleaner kills 99.9% germs, removes tough yellow stains and keeps toilet fresh.",No,No
`

	reader := csv.NewReader(bytes.NewReader([]byte(csvData)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	items := parseRecordsToImportItems(records)
	if len(items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(items))
	}

	// Verify Item 0: VIM Dish Bar Lemon (Pack of 3)
	vim := items[0]
	if vim.Name != "VIM Dish Bar Lemon (Pack of 3)" {
		t.Errorf("Expected Name 'VIM Dish Bar Lemon (Pack of 3)', got '%s'", vim.Name)
	}
	if vim.Price != 42.0 {
		t.Errorf("Expected Price 42.0, got %f", vim.Price)
	}
	if vim.CostPrice != 34.0 {
		t.Errorf("Expected CostPrice 34.0, got %f", vim.CostPrice)
	}
	if vim.ComparePrice != 45.0 {
		t.Errorf("Expected ComparePrice 45.0, got %f", vim.ComparePrice)
	}
	if vim.StockQuantity != 25 {
		t.Errorf("Expected StockQuantity 25, got %d", vim.StockQuantity)
	}
	if vim.MinStock != 5 {
		t.Errorf("Expected MinStock 5, got %d", vim.MinStock)
	}
	if vim.SKU != "VIMBAR03" {
		t.Errorf("Expected SKU 'VIMBAR03', got '%s'", vim.SKU)
	}
	if vim.CategoryName != "Household & Cleaning" {
		t.Errorf("Expected Category 'Household & Cleaning', got '%s'", vim.CategoryName)
	}
	if vim.Unit != "Piece" {
		t.Errorf("Expected Unit 'Piece', got '%s'", vim.Unit)
	}
	if !strings.Contains(vim.Description, "lemon power") {
		t.Errorf("Description should contain original text, got '%s'", vim.Description)
	}
	if !strings.Contains(vim.Description, "Fragrance: Lemon") {
		t.Errorf("Description should contain attributes, got '%s'", vim.Description)
	}

	// Verify Item 1: Surf Excel
	surf := items[1]
	if surf.Name != "Surf Excel Easy Wash Detergent Powder 1kg" {
		t.Errorf("Expected Name 'Surf Excel Easy Wash Detergent Powder 1kg', got '%s'", surf.Name)
	}
	if surf.Price != 115.0 {
		t.Errorf("Expected Price 115.0, got %f", surf.Price)
	}
	if surf.CostPrice != 98.0 {
		t.Errorf("Expected CostPrice 98.0, got %f", surf.CostPrice)
	}
	if surf.ComparePrice != 120.0 {
		t.Errorf("Expected ComparePrice 120.0, got %f", surf.ComparePrice)
	}
	if surf.StockQuantity != 15 {
		t.Errorf("Expected StockQuantity 15, got %d", surf.StockQuantity)
	}
}
