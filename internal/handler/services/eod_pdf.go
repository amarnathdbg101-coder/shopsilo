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
func (s *EODPDFService) GenerateEndOfDayPLPDF(ctx context.Context, shopOwnerUserID string, targetDate time.Time, days int) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shop for user %s: %w", shopOwnerUserID, err)
	}

	dateStr := targetDate.Format("2006-01-02")
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
	pdfBytes, err := RenderEODReportPDF(shop.Name, shopAddress, shopPhone, targetDate, summary)
	if err != nil {
		s.logger.Error("Failed to render native EOD PDF", zap.Error(err))
		return nil, err
	}

	duration := time.Since(start)
	s.logger.Info("Native gopdf End-of-Day P&L report generated successfully",
		zap.Int("bytes", len(pdfBytes)),
		zap.Duration("duration_ms", duration),
	)

	return pdfBytes, nil
}

// RenderEODReportPDF creates a clean, executive A4 Profit & Loss audit report using pure gopdf.
func RenderEODReportPDF(
	shopName, shopAddress, shopPhone string,
	targetDate time.Time,
	summary *dto.DailyCloseReport,
) ([]byte, error) {
	if summary == nil {
		summary = &dto.DailyCloseReport{}
	}

	formattedDate := targetDate.Format("02-Jan-2006")
	if summary.Date != "" && strings.Contains(summary.Date, "10-Day") {
		formattedDate = "Last 10 Days Rolling Window"
	}
	if strings.TrimSpace(shopName) == "" {
		shopName = "ShopSilo Partner Store"
	}

	totalSales := summary.TotalGrossSales
	expenses := summary.TotalExpensesPaid
	netProfit := totalSales - expenses
	if netProfit < 0 {
		netProfit = 0
	}

	cashShare := 0.0
	upiShare := 0.0
	creditShare := 0.0
	if totalSales > 0 {
		cashShare = (summary.CashSales / totalSales) * 100.0
		upiShare = (summary.UPISales / totalSales) * 100.0
		creditShare = (summary.CreditGiven / totalSales) * 100.0
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := utils.SetupInvoiceFonts(&pdf); err != nil {
		return nil, fmt.Errorf("failed to initialize embedded Noto Sans Devanagari fonts: %w", err)
	}

	pageW := 595.28
	margin := 36.0
	contentWidth := pageW - (margin * 2) // 523.28
	contentLeft := margin
	contentRight := contentLeft + contentWidth

	// Palette matching ShopSilo Executive Design System
	cNavyBanner := []uint8{15, 23, 42}       // #0F172A
	cHeaderSub := []uint8{148, 163, 184}     // #94A3B8
	cDarkText := []uint8{15, 23, 42}        // #0F172A
	cMutedText := []uint8{71, 85, 105}      // #475569
	cCardBg := []uint8{248, 250, 252}       // #F8FAFC
	cCardBorder := []uint8{226, 232, 240}   // #E2E8F0
	cEmeraldBg := []uint8{236, 253, 245}    // #ECFDF5
	cEmeraldBorder := []uint8{167, 243, 208}// #A7F3D0
	cEmeraldText := []uint8{4, 120, 87}     // #047857
	cTblHeader := []uint8{30, 41, 59}       // #1E293B
	cRedText := []uint8{220, 38, 38}        // #DC2626
	cAltRow := []uint8{248, 250, 252}       // #F8FAFC

	// Universal cell writer ensuring explicit non-zero bounding box on every single render
	writeCell := func(x, y, w, h float64, text string, fontName string, fontSize float64, color []uint8, align int) {
		pdf.SetTextColor(color[0], color[1], color[2])
		_ = pdf.SetFont(fontName, "", fontSize)
		pdf.SetXY(x, y)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w, H: h}, text, gopdf.CellOption{Align: align})
	}

	// 1. TOP EXECUTIVE DARK NAVY BANNER
	bannerH := 60.0
	pdf.SetFillColor(cNavyBanner[0], cNavyBanner[1], cNavyBanner[2])
	pdf.RectFromUpperLeftWithStyle(0, 0, pageW, bannerH, "F")

	// Store Name (bold, centered, prominent)
	writeCell(0, 10, pageW, 22, strings.ToUpper(shopName), "noto-bold", 16, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)

	// Subtitle in Banner
	writeCell(0, 34, pageW, 14, "EXECUTIVE FINANCIAL STATEMENT & INVENTORY AUDIT JOURNAL", "noto-bold", 8.5, cHeaderSub, gopdf.Center | gopdf.Middle)

	// 2. METADATA STRIP
	curY := bannerH + 10.0
	storeMeta := []string{}
	if strings.TrimSpace(shopAddress) != "" {
		storeMeta = append(storeMeta, strings.TrimSpace(shopAddress))
	}
	if strings.TrimSpace(shopPhone) != "" {
		storeMeta = append(storeMeta, "Phone: "+strings.TrimSpace(shopPhone))
	}
	if len(storeMeta) > 0 {
		writeCell(0, curY, pageW, 13, strings.Join(storeMeta, " • "), "noto-reg", 8.5, cMutedText, gopdf.Center | gopdf.Middle)
		curY += 14.0
	}

	// Audit Date Badge Line
	auditLine := fmt.Sprintf("Report Window: %s    |    Generated: %s", formattedDate, time.Now().Format("02-Jan-2006 15:04:05"))
	writeCell(0, curY, pageW, 14, auditLine, "noto-bold", 9, cDarkText, gopdf.Center | gopdf.Middle)
	curY += 20.0

	// 3. THREE FINANCIAL SUMMARY METRIC CARDS
	cardGap := 12.0
	cardW := (contentWidth - (cardGap * 2)) / 3.0
	cardH := 58.0

	// Card 1: Gross Sales Total
	x1 := contentLeft
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.7)
	pdf.RectFromUpperLeftWithStyle(x1, curY, cardW, cardH, "FD")

	writeCell(x1+10, curY+8, cardW-20, 12, "GROSS SALES TOTAL", "noto-bold", 8.5, cMutedText, gopdf.Left | gopdf.Middle)
	writeCell(x1+10, curY+22, cardW-20, 18, fmt.Sprintf("₹ %.2f", totalSales), "noto-bold", 13.5, cDarkText, gopdf.Left | gopdf.Middle)
	writeCell(x1+10, curY+41, cardW-20, 11, fmt.Sprintf("%d Bills Processed", summary.TotalBillsCount), "noto-reg", 8, cMutedText, gopdf.Left | gopdf.Middle)

	// Card 2: Expected Drawer Cash
	x2 := x1 + cardW + cardGap
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.RectFromUpperLeftWithStyle(x2, curY, cardW, cardH, "FD")

	writeCell(x2+10, curY+8, cardW-20, 12, "EXPECTED DRAWER CASH", "noto-bold", 8.5, cMutedText, gopdf.Left | gopdf.Middle)
	writeCell(x2+10, curY+22, cardW-20, 18, fmt.Sprintf("₹ %.2f", summary.ExpectedCashInDrawer), "noto-bold", 13.5, []uint8{49, 46, 129}, gopdf.Left | gopdf.Middle)
	writeCell(x2+10, curY+41, cardW-20, 11, "Physical Counter Cash", "noto-reg", 8, cMutedText, gopdf.Left | gopdf.Middle)

	// Card 3: Net Pocket Balance (Highlighted Emerald Green)
	x3 := x2 + cardW + cardGap
	pdf.SetFillColor(cEmeraldBg[0], cEmeraldBg[1], cEmeraldBg[2])
	pdf.SetStrokeColor(cEmeraldBorder[0], cEmeraldBorder[1], cEmeraldBorder[2])
	pdf.RectFromUpperLeftWithStyle(x3, curY, cardW, cardH, "FD")

	writeCell(x3+10, curY+8, cardW-20, 12, "NET POCKET PROFIT", "noto-bold", 8.5, cEmeraldText, gopdf.Left | gopdf.Middle)
	writeCell(x3+10, curY+22, cardW-20, 18, fmt.Sprintf("₹ %.2f", netProfit), "noto-bold", 13.5, cEmeraldText, gopdf.Left | gopdf.Middle)
	writeCell(x3+10, curY+41, cardW-20, 11, "After Store Expenses", "noto-reg", 8, cEmeraldText, gopdf.Left | gopdf.Middle)

	curY += cardH + 18.0

	// 4. COUNTER MULTI-TENDER BREAKDOWN TABLE
	writeCell(contentLeft, curY, contentWidth, 14, "Counter Multi-Tender Breakdown", "noto-bold", 10.5, cDarkText, gopdf.Left | gopdf.Middle)
	curY += 16.0

	// Table Header
	tblH := 22.0
	colModeW := 253.28
	colAmtW := 150.0
	colShareW := 120.0

	pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

	writeCell(contentLeft+10, curY, colModeW-10, tblH, "TENDER / PAYMENT MODE", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Left | gopdf.Middle)
	writeCell(contentLeft+colModeW, curY, colAmtW-10, tblH, "AMOUNT COLLECTED", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
	writeCell(contentLeft+colModeW+colAmtW, curY, colShareW-10, tblH, "SHARE %", "noto-bold", 8.5, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)

	curY += tblH

	type tenderRow struct {
		name      string
		amountStr string
		shareStr  string
		isExpense bool
	}

	rows := []tenderRow{
		{
			name:      "Physical Cash Tender",
			amountStr: fmt.Sprintf("₹ %.2f", summary.CashSales),
			shareStr:  fmt.Sprintf("%.1f%%", cashShare),
			isExpense: false,
		},
		{
			name:      "Instant UPI / Digital Pay",
			amountStr: fmt.Sprintf("₹ %.2f", summary.UPISales),
			shareStr:  fmt.Sprintf("%.1f%%", upiShare),
			isExpense: false,
		},
		{
			name:      "Khata Credit Given (Udhar)",
			amountStr: fmt.Sprintf("₹ %.2f", summary.CreditGiven),
			shareStr:  fmt.Sprintf("%.1f%%", creditShare),
			isExpense: false,
		},
		{
			name:      "Daily Store Operational Expenses",
			amountStr: fmt.Sprintf("- ₹ %.2f", expenses),
			shareStr:  "Deduction",
			isExpense: true,
		},
	}

	rowH := 20.0
	pdf.SetLineWidth(0.4)
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])

	for i, r := range rows {
		if i%2 == 1 {
			pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
			pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, rowH, "F")
		}

		pdf.Line(contentLeft, curY+rowH, contentRight, curY+rowH)

		writeCell(contentLeft+10, curY, colModeW-10, rowH, r.name, "noto-reg", 8.5, cDarkText, gopdf.Left | gopdf.Middle)

		amtColor := cDarkText
		if r.isExpense {
			amtColor = cRedText
		}
		writeCell(contentLeft+colModeW, curY, colAmtW-10, rowH, r.amountStr, "noto-bold", 8.5, amtColor, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+colModeW+colAmtW, curY, colShareW-10, rowH, r.shareStr, "noto-reg", 8, cMutedText, gopdf.Right | gopdf.Middle)

		curY += rowH
	}

	curY += 18.0

	// 5. CASH REGISTER RECONCILIATION BOX
	boxH := 64.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, boxH, "FD")

	writeCell(contentLeft+12, curY+6, contentWidth-24, 13, "Cash Register & POS Accountability Verification", "noto-bold", 9, cDarkText, gopdf.Left | gopdf.Middle)

	line1 := fmt.Sprintf("• Total Invoiced Bills: %d bills processed during reporting period.", summary.TotalBillsCount)
	writeCell(contentLeft+12, curY+22, contentWidth-24, 11, line1, "noto-reg", 8, cMutedText, gopdf.Left | gopdf.Middle)

	line2 := fmt.Sprintf("• Expected Closing Cash in Physical Drawer: ₹ %.2f (Calculated from Cash Sales + Khata In - Cash Expenses).", summary.ExpectedCashInDrawer)
	writeCell(contentLeft+12, curY+35, contentWidth-24, 11, line2, "noto-reg", 8, cMutedText, gopdf.Left | gopdf.Middle)

	line3 := "• Audit Verification: Reconciled automatically against ShopSilo cryptographic POS register journal."
	writeCell(contentLeft+12, curY+48, contentWidth-24, 11, line3, "noto-reg", 8, cMutedText, gopdf.Left | gopdf.Middle)

	curY += boxH + 20.0

	// 6. ITEMIZED SOLD PRODUCTS AUDIT JOURNAL TABLE
	if curY+50 > 750 {
		pdf.AddPage()
		curY = 36.0
	}

	writeCell(contentLeft, curY, contentWidth, 14, fmt.Sprintf("Itemized Sold Products Audit Journal (%d Items Sold)", len(summary.SoldItems)), "noto-bold", 10.5, cDarkText, gopdf.Left | gopdf.Middle)
	curY += 16.0

	// Column Widths
	colIdxW := 26.0
	colNameW := 217.28
	colQtyW := 45.0
	colRateW := 75.0
	colTotalW := 80.0
	colTimeW := 80.0

	drawItemTableHeader := func() {
		pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

		writeCell(contentLeft, curY, colIdxW, tblH, "#", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Center | gopdf.Middle)
		writeCell(contentLeft+colIdxW+4, curY, colNameW-4, tblH, "PRODUCT / ITEM NAME", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Left | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW, curY, colQtyW-4, tblH, "QTY", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW+colQtyW, curY, colRateW-4, tblH, "RATE (₹)", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW+colQtyW+colRateW, curY, colTotalW-4, tblH, "TOTAL (₹)", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW+colQtyW+colRateW+colTotalW, curY, colTimeW-4, tblH, "PAY / TIME", "noto-bold", 8, []uint8{255, 255, 255}, gopdf.Right | gopdf.Middle)

		curY += tblH
	}

	drawItemTableHeader()

	totalUnits := 0
	totalItemsRevenue := 0.0

	if len(summary.SoldItems) == 0 {
		pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, 24.0, "F")
		pdf.Line(contentLeft, curY+24.0, contentRight, curY+24.0)

		writeCell(contentLeft+10, curY, contentWidth-20, 24.0, "No individual counter bill items recorded for this reporting period.", "noto-reg", 8.5, cMutedText, gopdf.Left | gopdf.Middle)
		curY += 30.0
	} else {
		itemRowH := 18.0
		for i, item := range summary.SoldItems {
			totalUnits += item.Quantity
			totalItemsRevenue += item.TotalPrice

			// Page overflow guard
			if curY+itemRowH > 770 {
				pdf.AddPage()
				curY = 36.0

				writeCell(contentLeft, curY, contentWidth, 14, fmt.Sprintf("%s • Sold Products Journal (Contd.)", strings.ToUpper(shopName)), "noto-bold", 9.5, cDarkText, gopdf.Left | gopdf.Middle)
				curY += 16.0
				drawItemTableHeader()
			}

			if i%2 == 1 {
				pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
				pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, itemRowH, "F")
			}

			pdf.Line(contentLeft, curY+itemRowH, contentRight, curY+itemRowH)

			// #
			writeCell(contentLeft, curY, colIdxW, itemRowH, fmt.Sprintf("%d", i+1), "noto-reg", 8, cMutedText, gopdf.Center | gopdf.Middle)

			// Product Name (safe UTF-8 rune truncation)
			pName := utils.TruncateRunes(item.ProductName, 28)
			writeCell(contentLeft+colIdxW+4, curY, colNameW-4, itemRowH, pName, "noto-bold", 8.5, cDarkText, gopdf.Left | gopdf.Middle)

			// QTY
			writeCell(contentLeft+colIdxW+colNameW, curY, colQtyW-4, itemRowH, fmt.Sprintf("%d", item.Quantity), "noto-reg", 8.5, cDarkText, gopdf.Right | gopdf.Middle)

			// Unit Price
			writeCell(contentLeft+colIdxW+colNameW+colQtyW, curY, colRateW-4, itemRowH, fmt.Sprintf("₹ %.2f", item.UnitPrice), "noto-reg", 8, cMutedText, gopdf.Right | gopdf.Middle)

			// Total Price
			writeCell(contentLeft+colIdxW+colNameW+colQtyW+colRateW, curY, colTotalW-4, itemRowH, fmt.Sprintf("₹ %.2f", item.TotalPrice), "noto-bold", 8.5, cDarkText, gopdf.Right | gopdf.Middle)

			// Payment Mode & Time
			payMode := strings.ToUpper(item.PaymentMode)
			if payMode == "" {
				payMode = "CASH"
			}
			payTime := fmt.Sprintf("%s %s", payMode, item.SoldAt)
			writeCell(contentLeft+colIdxW+colNameW+colQtyW+colRateW+colTotalW, curY, colTimeW-4, itemRowH, payTime, "noto-reg", 7.5, cMutedText, gopdf.Right | gopdf.Middle)

			curY += itemRowH
		}

		// Total Row
		totalRowH := 20.0
		pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, totalRowH, "F")
		pdf.Line(contentLeft, curY+totalRowH, contentRight, curY+totalRowH)

		writeCell(contentLeft+colIdxW+4, curY, colNameW-4, totalRowH, fmt.Sprintf("TOTAL (%d PRODUCTS SOLD)", len(summary.SoldItems)), "noto-bold", 8.5, cDarkText, gopdf.Left | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW, curY, colQtyW-4, totalRowH, fmt.Sprintf("%d", totalUnits), "noto-bold", 8.5, cDarkText, gopdf.Right | gopdf.Middle)
		writeCell(contentLeft+colIdxW+colNameW+colQtyW+colRateW, curY, colTotalW-4, totalRowH, fmt.Sprintf("₹ %.2f", totalItemsRevenue), "noto-bold", 9, cEmeraldText, gopdf.Right | gopdf.Middle)

		curY += totalRowH + 16.0
	}

	if curY+45 > 760 {
		pdf.AddPage()
		curY = 36.0
	}

	// 7. CERTIFICATION STAMP & AUDIT SIGN-OFF
	writeCell(contentLeft, curY, contentWidth, 11, "This document serves as an immutable executive daily financial record generated by ShopSilo Operating System.", "noto-reg", 7.5, cMutedText, gopdf.Left | gopdf.Middle)
	curY += 12.0
	writeCell(contentLeft, curY, contentWidth, 11, "Authorized and verified for merchant accounting, tax auditing, and permanent records.", "noto-reg", 7.5, cMutedText, gopdf.Left | gopdf.Middle)

	// 8. FOOTER
	footerY := 805.0
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.5)
	pdf.Line(margin, footerY, contentRight, footerY)

	pdf.SetXY(contentLeft+colModeW, footerY+6)
	_ = pdf.CellWithOption(&gopdf.Rect{W: colAmtW + colShareW, H: 10}, "Confidential • Merchant Financial Audit", gopdf.CellOption{
		Align: gopdf.Right,
	})

	writeCell(margin, footerY+6, contentWidth, 10, "ShopSilo Retail OS • Pure Go Native PDF Engine • Immutable Digital Accounting Record", "noto-reg", 7.5, cMutedText, gopdf.Left | gopdf.Middle)

	return pdf.GetBytesPdfReturnErr()
}
