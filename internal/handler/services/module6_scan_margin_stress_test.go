package services

import (
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"strings"
	"testing"
)

func TestModule6ScanAndMargin_2000StressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(2026112806))

	// 1. 800 Scan & Go AddItemToCart validation cases
	for i := 0; i < 800; i++ {
		req := dto.AddItemToCartRequest{
			ProductID: fmt.Sprintf("p-%d", i),
			Barcode:   "",
			Quantity:  rng.Intn(600) - 50,
		}
		switch i % 6 {
		case 0:
			req.ProductID = "  "
			req.Barcode = ""
		case 1:
			req.ProductID = ""
			req.Barcode = "890123456"
		case 2:
			req.Quantity = 0
		case 3:
			req.Quantity = -5
		case 4:
			req.Quantity = 501
		}

		err := validateScanAddItemInput(&req)
		expectErr := (strings.TrimSpace(req.ProductID) == "" && strings.TrimSpace(req.Barcode) == "") ||
			req.Quantity < 0 || req.Quantity > 500
		if expectErr != (err != nil) {
			t.Fatalf("scan add item case %d: mismatch for %+v (err=%v)", i, req, err)
		}
		if err == nil && (req.Quantity < 1 || req.Quantity > 500) {
			t.Fatalf("scan add item case %d: normalized qty out of range: %d", i, req.Quantity)
		}
	}

	// 2. 600 Scan & Go ConvertToBill validation cases
	methods := []string{"", "CASH", "upi", "split", "SPLIT"}
	for i := 0; i < 600; i++ {
		req := dto.ConvertToBillRequest{
			PaymentMethod: methods[i%len(methods)],
			SplitCash:     float64(rng.Intn(500) - 50),
			SplitUPI:      float64(rng.Intn(500) - 50),
		}
		if i%10 == 0 {
			req.SplitCash = math.NaN()
		}
		if i%12 == 0 {
			req.SplitUPI = math.Inf(1)
		}

		err := validateConvertToBillInput(&req)
		pm := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
		badSplit := pm == "split" && (math.IsNaN(req.SplitCash) || math.IsInf(req.SplitCash, 0) || req.SplitCash < 0 ||
			math.IsNaN(req.SplitUPI) || math.IsInf(req.SplitUPI, 0) || req.SplitUPI < 0)
		if badSplit != (err != nil) {
			t.Fatalf("convert bill case %d: mismatch for %+v (err=%v)", i, req, err)
		}
	}

	// 3. 600 Margin Hotkey validation cases
	for i := 0; i < 600; i++ {
		req := dto.CreateShopHotkeyRequest{
			ProductID:   fmt.Sprintf("prod-%d", i),
			DisplayName: fmt.Sprintf("Item %d", i),
		}
		if i%5 == 0 {
			req.ProductID = "   "
		}
		if i%7 == 0 {
			req.DisplayName = ""
		}
		err := validateCreateHotkeyInput(&req)
		expectErr := strings.TrimSpace(req.ProductID) == "" || strings.TrimSpace(req.DisplayName) == ""
		if expectErr != (err != nil) {
			t.Fatalf("hotkey case %d: mismatch for %+v (err=%v)", i, req, err)
		}
	}
}
