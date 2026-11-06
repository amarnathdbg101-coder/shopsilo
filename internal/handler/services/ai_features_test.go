package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"

	"go.uber.org/zap"
)

func TestAIFeaturesSuite(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	shopRepo := repository.NewShopRepo(nil, logger)
	aiService := services.NewAIService(shopRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// =========================================================================
	// 1. Supplier Bill Photo-to-Inventory OCR Ingestion (FEAT-POS-02)
	// =========================================================================
	t.Run("Supplier Invoice OCR Ingestion", func(t *testing.T) {
		req := dto.ParseSupplierInvoiceRequest{
			ImageBase64:     "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=", // 1x1 dummy PNG
			MimeType:        "image/png",
			DistributorHint: "Varun Beverages Pepsico Distributor",
		}

		resp, err := aiService.ParseSupplierInvoice(ctx, req)
		if err != nil {
			t.Fatalf("ParseSupplierInvoice failed: %v", err)
		}
		if resp.DistributorName == "" {
			t.Errorf("Expected distributor name to be extracted")
		}
		if len(resp.Items) == 0 {
			t.Errorf("Expected extracted items in supplier invoice")
		} else {
			firstItem := resp.Items[0]
			t.Logf("Extracted Invoice: Distributor=%s, InvoiceNo=%s, Item=%s, Qty=%d, UnitCost=₹%.2f, SuggestedMRP=₹%.2f",
				resp.DistributorName, resp.InvoiceNumber, firstItem.ProductName, firstItem.Quantity, firstItem.UnitCost, firstItem.SuggestedMRP)
		}
	})

	// =========================================================================
	// 2. Real-Time POS Cross-Sell & Look-Stylist Recommendations (FEAT-SLS-01 & FEAT-FSH-02)
	// =========================================================================
	t.Run("POS Cross-Sell & Look-Stylist Recommendations", func(t *testing.T) {
		// A: Fashion + Cosmetic Synergy (Kurti -> Matching Lipstick)
		fashionReq := dto.AICrossSellRequest{
			ScannedProductName: "Embroidered Festive Anarkali Kurti (Blue)",
			CategoryHint:       "Women Ethnic Wear",
			Price:              1299.00,
		}
		fashionResp, err := aiService.GetCrossSellRecommendations(ctx, fashionReq)
		if err != nil {
			t.Fatalf("GetCrossSellRecommendations failed: %v", err)
		}
		if len(fashionResp.Recommendations) == 0 {
			t.Fatalf("Expected cross-sell recommendations for Kurti")
		}

		firstRec := fashionResp.Recommendations[0]
		if !strings.Contains(strings.ToLower(firstRec.Category), "cosmetics") && !strings.Contains(strings.ToLower(firstRec.SuggestedProductName), "lipstick") {
			t.Errorf("Expected cosmetic / lipstick recommendation for Kurti, got: %s", firstRec.SuggestedProductName)
		}
		if firstRec.CashierScriptHinglish == "" {
			t.Errorf("Expected cashier script in Hinglish")
		}
		t.Logf("Cloth+Cosmetic Synergy verified! Scanned='%s' -> Recommended='%s', MarginBoost=+₹%.2f, CashierScript='%s'",
			fashionResp.ScannedProduct, firstRec.SuggestedProductName, firstRec.MarginBoost, firstRec.CashierScriptHinglish)

		// B: FMCG Pairing (Chai Patti -> Premium Rusk)
		fmcgReq := dto.AICrossSellRequest{
			ScannedProductName: "Tata Tea Gold (500g)",
			CategoryHint:       "Groceries",
			Price:              320.00,
		}
		fmcgResp, err := aiService.GetCrossSellRecommendations(ctx, fmcgReq)
		if err != nil {
			t.Fatalf("GetCrossSellRecommendations failed: %v", err)
		}
		if len(fmcgResp.Recommendations) == 0 {
			t.Fatalf("Expected cross-sell recommendations for Tea")
		}
		t.Logf("FMCG pairing verified! Scanned='%s' -> Recommended='%s', CashierScript='%s'",
			fmcgResp.ScannedProduct, fmcgResp.Recommendations[0].SuggestedProductName, fmcgResp.Recommendations[0].CashierScriptHinglish)
	})

	// =========================================================================
	// 3. Dead-Stock Smart Bundler Copilot (FEAT-INV-02)
	// =========================================================================
	t.Run("Dead-Stock Smart Bundler Suggestions", func(t *testing.T) {
		bundleResp, err := aiService.GetDeadStockBundleSuggestions(ctx, "shop-test-101")
		if err != nil {
			t.Fatalf("GetDeadStockBundleSuggestions failed: %v", err)
		}
		if len(bundleResp.SuggestedBundles) == 0 {
			t.Fatalf("Expected at least 1 dead-stock liquidation bundle")
		}

		b := bundleResp.SuggestedBundles[0]
		if b.BundlePrice >= b.OriginalTotal {
			t.Errorf("Bundle price ₹%.2f should be less than original total ₹%.2f to offer customer savings", b.BundlePrice, b.OriginalTotal)
		}
		if b.EstimatedProfit <= 0 {
			t.Errorf("Bundle must yield positive profit above cost price")
		}
		t.Logf("Dead-stock bundle generated: '%s' (Slow: %s + Anchor: %s) -> BundlePrice=₹%.2f, Savings=₹%.2f, Profit=₹%.2f, Pitch='%s'",
			b.BundleName, b.SlowMovingItem, b.FastMovingAnchor, b.BundlePrice, b.CustomerSavings, b.EstimatedProfit, b.PitchLineHinglish)
	})

	// =========================================================================
	// 4. Psychology-Based Respectful WhatsApp Recovery Nudges (FEAT-KHT-04)
	// =========================================================================
	t.Run("Respectful Khata Recovery Nudges", func(t *testing.T) {
		nudgeReq := dto.AIRecoveryNudgeRequest{
			CustomerName:     "Kailash Sharma",
			CustomerPhone:    "9876543210",
			TotalDebt:        2400.00,
			DaysOverdue:      28,
			DailyMicroAmount: 60.00,
			ShopUPIID:        "bansalstore@okaxis",
			ShopName:         "Bansal General Store",
		}

		nudgeResp, err := aiService.GenerateSmartRecoveryNudge(ctx, nudgeReq)
		if err != nil {
			t.Fatalf("GenerateSmartRecoveryNudge failed: %v", err)
		}

		if !strings.Contains(nudgeResp.GentleNudge, "Kailash") || !strings.Contains(nudgeResp.GentleNudge, "2400.00") {
			t.Errorf("Gentle nudge missing customer name or debt amount")
		}
		if !strings.Contains(nudgeResp.MicroPlanNudge, "₹60") {
			t.Errorf("Micro-plan nudge should mention daily micro amount ₹60")
		}
		if !strings.Contains(nudgeResp.UPIPaymentLink, "bansalstore@okaxis") {
			t.Errorf("UPI payment link missing correct shop UPI ID")
		}

		t.Logf("Gentle Nudge: %s", nudgeResp.GentleNudge)
		t.Logf("Micro-Plan Nudge: %s", nudgeResp.MicroPlanNudge)
		t.Logf("Formal Notice: %s", nudgeResp.FormalNotice)
		t.Logf("UPI Pay Link: %s", nudgeResp.UPIPaymentLink)
	})
}
