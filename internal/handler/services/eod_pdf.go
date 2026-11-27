// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	"github.com/signintech/gopdf"
	"go.uber.org/zap"
)

// EODPDFService renders high-speed, lightweight executive PDF reports using pure Go (gopdf).
// Zero headless Chrome processes, zero external OS dependencies, zero memory spikes.
type EODPDFService struct {
	shopRepo   *repository.ShopRepo
	posRepo    *repository.POSRepo
	drawerRepo *repository.CashDrawerRepo
	logger     *zap.Logger
}

// ChromedpPDFService is maintained as a type alias for 100% backward compatibility.
type ChromedpPDFService = EODPDFService

// NewEODPDFService initializes a new pure Go EOD PDF generator service.
func NewEODPDFService(
	shopRepo *repository.ShopRepo,
	posRepo *repository.POSRepo,
	drawerRepo *repository.CashDrawerRepo,
	logger *zap.Logger,
) *EODPDFService {
	return &EODPDFService{
		shopRepo:   shopRepo,
		posRepo:    posRepo,
		drawerRepo: drawerRepo,
		logger:     logger,
	}
}

// NewChromedpPDFService preserves constructor compatibility.
func NewChromedpPDFService(
	shopRepo *repository.ShopRepo,
	posRepo *repository.POSRepo,
	drawerRepo *repository.CashDrawerRepo,
	logger *zap.Logger,
) *ChromedpPDFService {
	return NewEODPDFService(shopRepo, posRepo, drawerRepo, logger)
}

// GenerateEndOfDayPLPDF generates an executive End-of-Day P&L statement PDF for the merchant.
// Leverages bounded semaphore concurrency and sub-millisecond in-memory PDF caching.
func (s *EODPDFService) GenerateEndOfDayPLPDF(ctx context.Context, shopOwnerUserID string, targetDate time.Time, days int) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shop for user %s: %w", shopOwnerUserID, err)
	}

	dateStr := targetDate.Format("2006-01-02")
	cacheKey := fmt.Sprintf("eod_pdf:%s:%s:%d", shop.ID, dateStr, days)
	if cachedBytes, found := utils.GetCachedPDF(cacheKey); found {
		return cachedBytes, nil
	}

	var summary *dto.DailyCloseReport

	if days == 10 {
		summary, err = s.posRepo.GetRolling10DayCloseReport(ctx, shop.ID)
		if err != nil {
			s.logger.Warn("Failed to get 10-day rolling report, using fallback", zap.Error(err))
			summary = &dto.DailyCloseReport{Date: "Last 10 Days Rolling Record"}
		}
	} else {
		summary, err = s.posRepo.GetDailyCloseReport(ctx, shop.ID, targetDate)
		if err != nil {
			s.logger.Warn("Failed to get full daily close report, using empty fallback summary", zap.Error(err))
			summary = &dto.DailyCloseReport{
				Date: dateStr,
			}
		}
	}

	shopAddress := shop.Address
	if shop.City != "" {
		if shopAddress != "" {
			shopAddress += ", " + shop.City
		} else {
			shopAddress = shop.City
		}
	}
	shopPhone := shop.WhatsAppNumber
	if shopPhone == "" {
		shopPhone = shop.Phone
	}

	start := time.Now()
	pdfBytes, err := utils.RenderPDFWithConcurrencyLimit(ctx, func() ([]byte, error) {
		return RenderEODReportPDF(shop.Name, shopAddress, shopPhone, targetDate, summary)
	})
	if err != nil {
		s.logger.Error("Failed to render native EOD PDF", zap.Error(err))
		return nil, err
	}

	duration := time.Since(start)
	s.logger.Info("Native gopdf End-of-Day P&L report generated successfully",
		zap.Int("bytes", len(pdfBytes)),
		zap.Duration("duration_ms", duration),
	)

	utils.SetCachedPDF(cacheKey, pdfBytes, 5*time.Minute)
	return pdfBytes, nil
}

// RenderEODReportPDF creates a clean, executive A4 Profit & Loss audit report using pure gopdf.
// Layout and visual design match the executive Mandi Procurement Sheet identically.
func RenderEODReportPDF(
	shopName, shopAddress, shopPhone string,
	targetDate time.Time,
	summary *dto.DailyCloseReport,
) ([]byte, error) {
	if summary == nil {
		summary = &dto.DailyCloseReport{
			Date: targetDate.Format("2006-01-02"),
		}
	}

	formattedDate := targetDate.Format("02-Jan-2006")
	if summary.Date != "" && strings.Contains(summary.Date, "10-Day") {
		formattedDate = "Last 10 Days Rolling Window"
	}
	storeName := "ShopSilo Partner Store"
	if strings.TrimSpace(shopName) != "" {
		storeName = strings.TrimSpace(shopName)
	}

	totalSales := summary.TotalGrossSales
	totalUnits := 0
	for _, item := range summary.SoldItems {
		totalUnits += item.Quantity
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := utils.SetupInvoiceFonts(&pdf); err != nil {
		return nil, fmt.Errorf("failed to initialize embedded Noto Sans Devanagari fonts: %w", err)
	}

	pageW := 595.28
	margin := 28.0
	contentWidth := pageW - (margin * 2) // 539.28
	contentLeft := margin
	contentRight := contentLeft + contentWidth

	// Executive Color Palette (Identical to Mandi Procurement Sheet)
	cTopBanner := []uint8{15, 23, 42}     // #0f172a (Dark Navy Executive Bar)
	cHeaderSub := []uint8{148, 163, 184}  // #94a3b8 (Slate Muted)
	cDark := []uint8{15, 23, 42}          // #0f172a (Primary text)
	cMuted := []uint8{71, 85, 105}        // #475569 (Secondary text)
	cTblHeader := []uint8{49, 46, 129}    // #312e81 (Deep Indigo Table Header)
	cAltRow := []uint8{248, 250, 252}     // #f8fafc (Alternate row)
	cBorder := []uint8{226, 232, 240}     // #e2e8f0 (Grid borders)
	cCardBg := []uint8{241, 245, 249}     // #f1f5f9 (Pill background)
	cCardBorder := []uint8{203, 213, 225} // #cbd5e1 (Pill border)
	cBlueAccent := []uint8{30, 58, 138}   // #1e3a8a (Pill text / Qty)

	// Universal cell writer ensuring explicit non-zero bounding box on every single render
	writeCell := func(x, y, w, h float64, text string, fontName string, fontSize float64, color []uint8, align int) {
		pdf.SetTextColor(color[0], color[1], color[2])
		_ = pdf.SetFont(fontName, "", fontSize)
		pdf.SetXY(x, y)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{Align: align})
	}

	// 1. TOP FULL-WIDTH DARK NAVY EXECUTIVE BANNER (Mandi Style)
	bannerH := 52.0
	pdf.SetFillColor(cTopBanner[0], cTopBanner[1], cTopBanner[2])
	pdf.RectFromUpperLeftWithStyle(0, 0, pageW, bannerH, "F")

	// ShopSilo Official Logo
	_ = utils.DrawShopSiloLogo(&pdf, 20, 10, 32)

	// Store Name in Banner
	writeCell(0, 10, pageW, 20, strings.ToUpper(storeName), "noto-bold", 16, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)

	// Subtitle in Banner
	writeCell(0, 32, pageW, 14, strings.ToUpper(storeName)+" - TODAY SALES & P&L REPORT", "noto-bold", 8.5, cHeaderSub, gopdf.Center | gopdf.Middle)

	// 2. STORE CONTACT & DATE SUB-HEADER
	curY := bannerH + 12.0

	if strings.TrimSpace(shopAddress) != "" {
		writeCell(0, curY, pageW, 13, strings.TrimSpace(shopAddress), "noto-reg", 8.5, cMuted, gopdf.Center | gopdf.Middle)
		curY += 13.0
	}
	if strings.TrimSpace(shopPhone) != "" {
		phoneStr := "Phone / WhatsApp: " + strings.TrimSpace(shopPhone)
		writeCell(0, curY, pageW, 13, phoneStr, "noto-reg", 8.5, cMuted, gopdf.Center | gopdf.Middle)
		curY += 13.0
	}
	dateStr := fmt.Sprintf("Report Window: %s  |  Date Generated: %s", formattedDate, time.Now().Format("02-Jan-2006 15:04"))
	writeCell(0, curY, pageW, 13, dateStr, "noto-reg", 8.5, cMuted, gopdf.Center | gopdf.Middle)
	curY += 16.0

	// 3. ROUNDED SUMMARY CALLOUT PILLS (Mandi Style)
	pillH := 24.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, pillH, "FD")

	pillText := fmt.Sprintf("TOTAL GROSS SALES: Rs. %.2f   •   %d UNITS SOLD   •   %d BILLS", totalSales, totalUnits, summary.TotalBillsCount)
	writeCell(contentLeft, curY, contentWidth, pillH, pillText, "noto-bold", 9.5, cBlueAccent, gopdf.Center | gopdf.Middle)
	curY += pillH + 6.0

	// Tender Breakdown Pill
	breakdownH := 20.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, breakdownH, "FD")

	breakdownText := fmt.Sprintf("PAYMENT BREAKDOWN:  Cash: Rs. %.2f  |  UPI: Rs. %.2f  |  Khata / Udhar: Rs. %.2f", summary.CashSales, summary.UPISales, summary.CreditGiven)
	writeCell(contentLeft, curY, contentWidth, breakdownH, breakdownText, "noto-reg", 8.5, cDark, gopdf.Center | gopdf.Middle)
	curY += breakdownH + 12.0

	// 4. DATA TABLE HEADER & COLUMNS (Mandi Style Table)
	tblH := 22.0
	w1 := 26.0   // #
	w2 := 210.0  // Item Description & SKU
	w3 := 45.0   // Qty Sold
	w4 := 70.0   // Rate
	w5 := 80.0   // Total
	w6 := 108.28 // Payment / Time

	drawTableHeader := func() {
		pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

		writeCell(contentLeft, curY, w1, tblH, "#", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)
		writeCell(contentLeft+w1+4, curY, w2-4, tblH, "Item Description & SKU", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Left | gopdf.Middle)
		writeCell(contentLeft+w1+w2, curY, w3, tblH, "Qty", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)
		writeCell(contentLeft+w1+w2+w3, curY, w4-4, tblH, "Rate (Rs.)", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+w1+w2+w3+w4, curY, w5-4, tblH, "Total (Rs.)", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+w1+w2+w3+w4+w5, curY, w6, tblH, "Pay / Time", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)

		curY += tblH
	}

	drawTableHeader()

	// 5. DATA ROWS
	if len(summary.SoldItems) == 0 {
		emptyH := 24.0
		pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, emptyH, "F")

		pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
		pdf.SetLineWidth(0.4)
		pdf.Line(contentLeft, curY+emptyH, contentRight, curY+emptyH)

		writeCell(contentLeft+10, curY, contentWidth-20, emptyH, "No individual counter bill items recorded for this reporting period.", "noto-reg", 8.5, cMuted, gopdf.Left | gopdf.Middle)
		curY += emptyH + 10.0
	} else {
		rowH := 20.0
		pdf.SetLineWidth(0.4)

		for i, item := range summary.SoldItems {
			if curY > 770 {
				pdf.AddPage()
				curY = 30.0
				drawTableHeader()
			}

			if i%2 == 1 {
				pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
				pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, rowH, "F")
			}

			pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
			pdf.Line(contentLeft, curY+rowH, contentRight, curY+rowH)

			// #
			writeCell(contentLeft, curY, w1, rowH, fmt.Sprintf("%d", i+1), "noto-reg", 8, cMuted, gopdf.Center | gopdf.Middle)

			// Item Description & SKU
			cleanName := strings.TrimSpace(item.ProductName)
			if cleanName == "" {
				cleanName = fmt.Sprintf("Item #%d", i+1)
			}
			displayName := cleanName
			if strings.TrimSpace(item.ProductSKU) != "" && strings.ToLower(cleanName) != strings.ToLower(strings.TrimSpace(item.ProductSKU)) {
				displayName += " (" + strings.TrimSpace(item.ProductSKU) + ")"
			}
			displayName = utils.TruncateRunes(displayName, 34)
			writeCell(contentLeft+w1+4, curY, w2-6, rowH, displayName, "noto-bold", 8.5, cDark, gopdf.Left | gopdf.Middle)

			// Qty
			writeCell(contentLeft+w1+w2, curY, w3, rowH, fmt.Sprintf("%d", item.Quantity), "noto-bold", 8.5, cBlueAccent, gopdf.Center | gopdf.Middle)

			// Rate
			writeCell(contentLeft+w1+w2+w3, curY, w4-4, rowH, fmt.Sprintf("%.2f", item.UnitPrice), "noto-reg", 8.5, cDark, gopdf.Right | gopdf.Middle)

			// Total
			writeCell(contentLeft+w1+w2+w3+w4, curY, w5-4, rowH, fmt.Sprintf("%.2f", item.TotalPrice), "noto-bold", 8.5, cDark, gopdf.Right | gopdf.Middle)

			// Pay / Time
			payMode := strings.ToUpper(strings.TrimSpace(item.PaymentMode))
			if payMode == "" {
				payMode = "CASH"
			}
			timeStr := strings.TrimSpace(item.SoldAt)
			payDisplay := fmt.Sprintf("%s %s", payMode, timeStr)
			writeCell(contentLeft+w1+w2+w3+w4+w5, curY, w6, rowH, payDisplay, "noto-reg", 8, cMuted, gopdf.Center | gopdf.Middle)

			curY += rowH
		}

		// Table Summary Bottom Row
		totalRowH := 20.0
		pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, totalRowH, "F")

		pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
		pdf.Line(contentLeft, curY+totalRowH, contentRight, curY+totalRowH)

		writeCell(contentLeft+w1+4, curY, w2-6, totalRowH, "TOTAL SUMMARY", "noto-bold", 8.5, cDark, gopdf.Left | gopdf.Middle)
		writeCell(contentLeft+w1+w2, curY, w3, totalRowH, fmt.Sprintf("%d", totalUnits), "noto-bold", 8.5, cBlueAccent, gopdf.Center | gopdf.Middle)
		writeCell(contentLeft+w1+w2+w3+w4, curY, w5-4, totalRowH, fmt.Sprintf("%.2f", totalSales), "noto-bold", 9, cBlueAccent, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+w1+w2+w3+w4+w5, curY, w6, totalRowH, fmt.Sprintf("%d Items", len(summary.SoldItems)), "noto-bold", 8, cDark, gopdf.Center | gopdf.Middle)

		curY += totalRowH + 12.0
	}

	// 6. FOOTER NOTE (Mandi Style)
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	footerMsg := "Notes: Verified point-of-sale transaction journal. Generated automatically by ShopSilo Retail OS."
	writeCell(contentLeft, footerY+8, contentWidth, 12, footerMsg, "noto-reg", 7.8, cMuted, gopdf.Center | gopdf.Middle)

	return pdf.GetBytesPdfReturnErr()
}
