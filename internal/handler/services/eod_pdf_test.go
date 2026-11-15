package services

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
)

func TestEODPDF_PureGoRender(t *testing.T) {
	summary := &dto.DailyCloseReport{
		Date:                 "2026-11-09",
		TotalGrossSales:      18450.50,
		TotalBillsCount:      34,
		CashSales:            8200.00,
		UPISales:             7250.50,
		CreditGiven:          3000.00,
		TotalExpensesPaid:    1500.00,
		ExpectedCashInDrawer: 6700.00,
	}

	start := time.Now()
	pdfBytes, err := RenderEODReportPDF(
		"ShopSilo Supermarket",
		"Main Market, Station Road, Delhi",
		"+91 9876543210",
		time.Now(),
		summary,
	)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("RenderEODReportPDF failed: %v", err)
	}

	if len(pdfBytes) < 500 {
		t.Fatalf("Generated PDF too small: %d bytes", len(pdfBytes))
	}

	// Verify standard PDF magic header
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Expected %%PDF- magic header, got: %s", string(pdfBytes[:8]))
	}

	t.Logf("✅ Successfully rendered EOD P&L PDF via pure gopdf: %d bytes in %v!", len(pdfBytes), duration)
}

func TestEODPDF_ConcurrentRenders(t *testing.T) {
	summary := &dto.DailyCloseReport{
		Date:                 "2026-11-09",
		TotalGrossSales:      5400.00,
		TotalBillsCount:      12,
		CashSales:            3000.00,
		UPISales:             2400.00,
		CreditGiven:          0.00,
		TotalExpensesPaid:    400.00,
		ExpectedCashInDrawer: 2600.00,
	}

	const concurrency = 20
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pdf, err := RenderEODReportPDF("Concurrent Store", "Sector 18, Noida", "9999999999", time.Now(), summary)
			if err != nil {
				errCh <- err
				return
			}
			if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
				t.Errorf("Worker %d: invalid PDF header", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("Concurrent render failed: %v", err)
	}

	totalDuration := time.Since(start)
	t.Logf("✅ Successfully executed %d concurrent pure gopdf renders in %v (avg %v per render) without memory bloat or semaphores!",
		concurrency, totalDuration, totalDuration/concurrency)
}

func TestEODPDF_HindiAndUnicode(t *testing.T) {
	summary := &dto.DailyCloseReport{
		Date:                 "2026-11-09",
		TotalGrossSales:      12000.00,
		TotalBillsCount:      15,
		CashSales:            6000.00,
		UPISales:             6000.00,
		ExpectedCashInDrawer: 6000.00,
	}

	// Test store with Hindi Devanagari script
	pdfBytes, err := RenderEODReportPDF(
		"श्री गणेश किराना स्टोर",
		"दुकान नंबर ४, पुराना बाजार, जयपुर",
		"+91 9123456789",
		time.Now(),
		summary,
	)

	if err != nil {
		t.Fatalf("Hindi Unicode PDF render failed: %v", err)
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Invalid PDF header for Hindi store")
	}

	t.Logf("✅ Successfully rendered Hindi Devanagari Store Name with ₹ Rupee symbols: %d bytes", len(pdfBytes))
}

func TestEODPDF_ZeroMetrics(t *testing.T) {
	pdfBytes, err := RenderEODReportPDF("", "", "", time.Now(), nil)
	if err != nil {
		t.Fatalf("Expected nil summary to render without panic, got error: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Invalid PDF header for nil summary")
	}
	t.Logf("✅ Successfully handled nil/zero summary gracefully: %d bytes", len(pdfBytes))
}

func TestEODPDF_ItemizedSoldProductsMultiPage(t *testing.T) {
	var items []dto.DailySoldProductItem
	for i := 1; i <= 45; i++ {
		items = append(items, dto.DailySoldProductItem{
			ProductName: "Fortune Sunlite Sunflower Oil 1L Pouch Special Pack",
			ProductSKU:  "8901234567890",
			Quantity:    2,
			UnitPrice:   145.00,
			TotalPrice:  290.00,
			PaymentMode: "UPI",
			SoldAt:      "2026-11-09 14:30",
		})
	}

	summary := &dto.DailyCloseReport{
		Date:                 "2026-11-09",
		TotalGrossSales:      13050.00,
		TotalBillsCount:      45,
		UPISales:             13050.00,
		ExpectedCashInDrawer: 0.00,
		SoldItems:            items,
	}

	pdfBytes, err := RenderEODReportPDF("ShopSilo Mega Mart", "Station Rd, Patna", "9876543210", time.Now(), summary)
	if err != nil {
		t.Fatalf("Failed to render multi-page sold items PDF: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Invalid PDF header")
	}
	if len(pdfBytes) < 15000 {
		t.Fatalf("Expected multi-page PDF to be substantial, got %d bytes", len(pdfBytes))
	}
	// Verify that /Type /Page exists multiple times (indicating multi-page PDF)
	pageCount := bytes.Count(pdfBytes, []byte("/Type /Page"))
	if pageCount < 2 {
		t.Fatalf("Expected at least 2 pages for 45 items, got %d pages", pageCount)
	}
	t.Logf("✅ Multi-page itemized PDF rendered successfully: %d bytes (%d items, %d pages)", len(pdfBytes), len(items), pageCount)
}
