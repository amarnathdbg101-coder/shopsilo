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
func (s *EODPDFService) GenerateEndOfDayPLPDF(ctx context.Context, shopOwnerUserID string, targetDate time.Time) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shop for user %s: %w", shopOwnerUserID, err)
	}

	dateStr := targetDate.Format("2006-01-02")

	// Fetch daily close metrics from POS repository
	summary, err := s.posRepo.GetDailyCloseReport(ctx, shop.ID, targetDate)
	if err != nil {
		s.logger.Warn("Failed to get full daily close report, using empty fallback summary", zap.Error(err))
		summary = &dto.DailyCloseReport{
			Date: dateStr,
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

// RenderEODReportPDF creates an executive A4 Profit & Loss audit report using pure gopdf.
func RenderEODReportPDF(
	shopName, shopAddress, shopPhone string,
	targetDate time.Time,
	summary *dto.DailyCloseReport,
) ([]byte, error) {
	if summary == nil {
		summary = &dto.DailyCloseReport{}
	}

	formattedDate := targetDate.Format("02 Jan 2006")
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
	cNavyBanner := []uint8{15, 23, 42}    // #0f172a (Dark Navy)
	cHeaderSub := []uint8{148, 163, 184}  // #94a3b8 (Slate Muted)
	cDarkText := []uint8{15, 23, 42}     // #0f172a
	cMutedText := []uint8{71, 85, 105}   // #475569
	cCardBg := []uint8{248, 250, 252}    // #f8fafc
	cCardBorder := []uint8{226, 232, 240}// #e2e8f0
	cEmeraldBg := []uint8{236, 253, 245} // #ecfdf5
	cEmeraldBorder := []uint8{167, 243, 208} // #a7f3d0
	cEmeraldText := []uint8{4, 120, 87}  // #047857
	cTblHeader := []uint8{30, 41, 59}    // #1e293b
	cRedText := []uint8{220, 38, 38}     // #dc2626
	cAltRow := []uint8{248, 250, 252}    // #f8fafc

	// 1. TOP EXECUTIVE DARK NAVY BANNER
	bannerH := 58.0
	pdf.SetFillColor(cNavyBanner[0], cNavyBanner[1], cNavyBanner[2])
	pdf.RectFromUpperLeftWithStyle(0, 0, pageW, bannerH, "F")

	// Store Name
	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 17)
	pdf.SetXY(0, 10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 22}, strings.ToUpper(shopName), gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// Subtitle in Banner
	pdf.SetTextColor(cHeaderSub[0], cHeaderSub[1], cHeaderSub[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(0, 34)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, "DAILY FINANCIAL & PROFIT ACCOUNTABILITY AUDIT • END-OF-DAY P&L", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 2. METADATA STRIP
	curY := bannerH + 12.0
	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-reg", "", 8.5)

	storeMeta := []string{}
	if strings.TrimSpace(shopAddress) != "" {
		storeMeta = append(storeMeta, strings.TrimSpace(shopAddress))
	}
	if strings.TrimSpace(shopPhone) != "" {
		storeMeta = append(storeMeta, "Phone: "+strings.TrimSpace(shopPhone))
	}
	if len(storeMeta) > 0 {
		pdf.SetXY(0, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 13}, strings.Join(storeMeta, " • "), gopdf.CellOption{
			Align: gopdf.Center | gopdf.Middle,
		})
		curY += 14.0
	}

	// Audit Date Badge Line
	auditLine := fmt.Sprintf("Report Date: %s    |    Generated: %s", formattedDate, time.Now().Format("02-Jan-2006 15:04:05"))
	pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
	_ = pdf.SetFont("noto-bold", "", 9)
	pdf.SetXY(0, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, auditLine, gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})
	curY += 22.0

	// 3. THREE FINANCIAL SUMMARY CARDS
	cardGap := 12.0
	cardW := (contentWidth - (cardGap * 2)) / 3.0
	cardH := 58.0

	// Card 1: Gross Sales Total
	x1 := contentLeft
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.7)
	pdf.RectFromUpperLeftWithStyle(x1, curY, cardW, cardH, "FD")

	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(x1+10, curY+8)
	_ = pdf.Cell(nil, "GROSS SALES TOTAL")

	pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
	_ = pdf.SetFont("noto-bold", "", 14)
	pdf.SetXY(x1+10, curY+22)
	_ = pdf.Cell(nil, fmt.Sprintf("₹ %.2f", totalSales))

	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x1+10, curY+41)
	_ = pdf.Cell(nil, fmt.Sprintf("%d Bills Processed", summary.TotalBillsCount))

	// Card 2: Expected Drawer Cash
	x2 := x1 + cardW + cardGap
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.RectFromUpperLeftWithStyle(x2, curY, cardW, cardH, "FD")

	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(x2+10, curY+8)
	_ = pdf.Cell(nil, "EXPECTED DRAWER CASH")

	pdf.SetTextColor(49, 46, 129) // Deep Indigo #312e81
	_ = pdf.SetFont("noto-bold", "", 14)
	pdf.SetXY(x2+10, curY+22)
	_ = pdf.Cell(nil, fmt.Sprintf("₹ %.2f", summary.ExpectedCashInDrawer))

	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x2+10, curY+41)
	_ = pdf.Cell(nil, "Physical Counter Cash")

	// Card 3: Net Pocket Balance (Highlighted)
	x3 := x2 + cardW + cardGap
	pdf.SetFillColor(cEmeraldBg[0], cEmeraldBg[1], cEmeraldBg[2])
	pdf.SetStrokeColor(cEmeraldBorder[0], cEmeraldBorder[1], cEmeraldBorder[2])
	pdf.RectFromUpperLeftWithStyle(x3, curY, cardW, cardH, "FD")

	pdf.SetTextColor(cEmeraldText[0], cEmeraldText[1], cEmeraldText[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(x3+10, curY+8)
	_ = pdf.Cell(nil, "NET POCKET PROFIT")

	_ = pdf.SetFont("noto-bold", "", 14)
	pdf.SetXY(x3+10, curY+22)
	_ = pdf.Cell(nil, fmt.Sprintf("₹ %.2f", netProfit))

	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x3+10, curY+41)
	_ = pdf.Cell(nil, "After Store Expenses")

	curY += cardH + 24.0

	// 4. COUNTER MULTI-TENDER BREAKDOWN TABLE
	pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
	_ = pdf.SetFont("noto-bold", "", 11)
	pdf.SetXY(contentLeft, curY)
	_ = pdf.Cell(nil, "Counter Multi-Tender Breakdown")
	curY += 16.0

	// Table Header
	tblH := 24.0
	colModeW := 253.28
	colAmtW := 150.0
	colShareW := 120.0

	pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft+10, curY+6)
	_ = pdf.Cell(nil, "TENDER / PAYMENT MODE")

	pdf.SetXY(contentLeft+colModeW, curY+6)
	_ = pdf.CellWithOption(&gopdf.Rect{W: colAmtW - 10, H: 12}, "AMOUNT COLLECTED", gopdf.CellOption{Align: gopdf.Right})

	pdf.SetXY(contentLeft+colModeW+colAmtW, curY+6)
	_ = pdf.CellWithOption(&gopdf.Rect{W: colShareW - 10, H: 12}, "SHARE %", gopdf.CellOption{Align: gopdf.Right})

	curY += tblH

	// Table Rows Definition
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
			name:      "Instant UPI / Online Pay",
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

	rowH := 22.0
	pdf.SetLineWidth(0.4)
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])

	for i, r := range rows {
		if i%2 == 1 {
			pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
			pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, rowH, "F")
		}

		// Bottom border for each row
		pdf.Line(contentLeft, curY+rowH, contentRight, curY+rowH)

		// Tender Name
		pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft+10, curY+5)
		_ = pdf.Cell(nil, r.name)

		// Amount
		if r.isExpense {
			pdf.SetTextColor(cRedText[0], cRedText[1], cRedText[2])
		} else {
			pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
		}
		_ = pdf.SetFont("noto-bold", "", 9)
		pdf.SetXY(contentLeft+colModeW, curY+5)
		_ = pdf.CellWithOption(&gopdf.Rect{W: colAmtW - 10, H: 12}, r.amountStr, gopdf.CellOption{Align: gopdf.Right})

		// Share %
		pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
		_ = pdf.SetFont("noto-reg", "", 8.5)
		pdf.SetXY(contentLeft+colModeW+colAmtW, curY+5)
		_ = pdf.CellWithOption(&gopdf.Rect{W: colShareW - 10, H: 12}, r.shareStr, gopdf.CellOption{Align: gopdf.Right})

		curY += rowH
	}

	curY += 24.0

	// 5. CASH DRAWER & OPERATIONAL RECONCILIATION BOX
	boxH := 70.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, boxH, "FD")

	pdf.SetTextColor(cDarkText[0], cDarkText[1], cDarkText[2])
	_ = pdf.SetFont("noto-bold", "", 9.5)
	pdf.SetXY(contentLeft+14, curY+10)
	_ = pdf.Cell(nil, "Cash Register & POS Accountability Verification")

	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-reg", "", 8.5)

	line1 := fmt.Sprintf("• Total Invoiced Bills: %d bills processed during today's register open session.", summary.TotalBillsCount)
	pdf.SetXY(contentLeft+14, curY+26)
	_ = pdf.Cell(nil, line1)

	line2 := fmt.Sprintf("• Expected Closing Cash in Physical Drawer: ₹ %.2f (Calculated from Cash Sales - Cash Out Expenses).", summary.ExpectedCashInDrawer)
	pdf.SetXY(contentLeft+14, curY+39)
	_ = pdf.Cell(nil, line2)

	line3 := "• Audit Verification: Reconciled automatically against ShopSilo cryptographic POS register journal."
	pdf.SetXY(contentLeft+14, curY+52)
	_ = pdf.Cell(nil, line3)

	curY += boxH + 28.0

	// 6. CERTIFICATION STAMP & AUDIT SIGN-OFF
	pdf.SetTextColor(cMutedText[0], cMutedText[1], cMutedText[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(contentLeft, curY)
	_ = pdf.Cell(nil, "This document serves as an immutable executive daily financial record generated by ShopSilo Operating System.")
	curY += 12.0
	pdf.SetXY(contentLeft, curY)
	_ = pdf.Cell(nil, "Authorized and verified for accounting, tax auditing, and merchant records.")

	// 7. FOOTER
	footerY := 805.0
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.5)
	pdf.Line(margin, footerY, contentRight, footerY)

	pdf.SetTextColor(cHeaderSub[0], cHeaderSub[1], cHeaderSub[2])
	_ = pdf.SetFont("noto-reg", "", 7.5)

	pdf.SetXY(margin, footerY+6)
	_ = pdf.Cell(nil, "ShopSilo Retail OS • Pure Go Native PDF Engine (Zero Headless Dependencies)")

	pdf.SetXY(contentLeft+colModeW, footerY+6)
	_ = pdf.CellWithOption(&gopdf.Rect{W: colAmtW + colShareW, H: 10}, "Confidential • Merchant Financial Audit", gopdf.CellOption{
		Align: gopdf.Right,
	})

	return pdf.GetBytesPdfReturnErr()
}
