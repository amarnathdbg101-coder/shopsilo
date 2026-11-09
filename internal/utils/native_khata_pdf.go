// Package utils provides high-performance native PDF generators for ShopSilo.
package utils

import (
	"fmt"
	"strings"
	"time"

	gohumanize "github.com/amarnathdbg101-coder/go/go-humanize"
	"github.com/skip2/go-qrcode"
	"github.com/signintech/gopdf"
	"shopMe/internal/handler/model"
)

// GenerateNativeKhataStatementPDF builds an executive A4 customer passbook statement in pure Go.
func GenerateNativeKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := setupInvoiceFonts(&pdf); err != nil {
		return nil, err
	}

	pageW := 595.28
	margin := 28.0
	contentWidth := pageW - (margin * 2) // 539.28
	contentLeft := margin
	contentRight := contentLeft + contentWidth

	// Executive Color Palette matching ShopSilo Design System
	cTopBanner := []uint8{15, 23, 42}    // #0f172a (Dark Navy Executive Bar)
	cHeaderSub := []uint8{148, 163, 184}  // #94a3b8 (Slate Muted)
	cDark := []uint8{15, 23, 42}         // #0f172a (Primary text)
	cMuted := []uint8{71, 85, 105}       // #475569 (Secondary text)
	cTblHeader := []uint8{49, 46, 129}   // #312e81 (Deep Indigo Table Header)
	cAltRow := []uint8{248, 250, 252}    // #f8fafc (Alternate row)
	cBorder := []uint8{226, 232, 240}    // #e2e8f0 (Grid borders)
	cRed := []uint8{220, 38, 38}         // #dc2626 (Udhar / Debit)
	cGreen := []uint8{22, 163, 74}       // #16a34a (Payment / Credit)
	cCardBg := []uint8{248, 250, 252}    // #f8fafc

	storeName := "ShopSilo Partner Store"
	storeAddress := ""
	storePhone := ""
	storeUPI := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			storeName = strings.TrimSpace(shop.Name)
		}
		addrParts := []string{}
		if strings.TrimSpace(shop.Address) != "" {
			addrParts = append(addrParts, strings.TrimSpace(shop.Address))
		}
		if strings.TrimSpace(shop.City) != "" {
			cityStr := shop.City
			if shop.Pincode != "" {
				cityStr += " " + shop.Pincode
			}
			addrParts = append(addrParts, cityStr)
		}
		storeAddress = strings.Join(addrParts, ", ")

		if shop.WhatsAppNumber != "" {
			storePhone = "Phone / WhatsApp: " + shop.WhatsAppNumber
		} else if shop.Phone != "" {
			storePhone = "Phone: " + shop.Phone
		}
		storeUPI = shop.UPIID
	}

	custName := "Customer"
	custPhone := ""
	currentDue := 0.0
	if customer != nil {
		if strings.TrimSpace(customer.CustomerName) != "" {
			custName = strings.TrimSpace(customer.CustomerName)
		}
		custPhone = maskPhone(customer.CustomerMobile)
		currentDue = customer.CurrentBalance
	}

	// Calculate totals
	totalDebit := 0.0
	totalCredit := 0.0
	for _, tx := range transactions {
		if tx.Type == model.KhataTxTypeGiveCredit || tx.Type == "debit" || tx.Type == "DEBIT" {
			totalDebit += tx.Amount
		} else {
			totalCredit += tx.Amount
		}
	}

	// 1. TOP FULL-WIDTH DARK NAVY EXECUTIVE BANNER
	bannerH := 52.0
	pdf.SetFillColor(cTopBanner[0], cTopBanner[1], cTopBanner[2])
	pdf.RectFromUpperLeftWithStyle(0, 0, pageW, bannerH, "F")

	// Store Name in Banner
	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 16)
	pdf.SetXY(0, 10)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 20}, strings.ToUpper(storeName), gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// Subtitle in Banner
	pdf.SetTextColor(cHeaderSub[0], cHeaderSub[1], cHeaderSub[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(0, 32)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, "OFFICIAL CUSTOMER KHATA PASSBOOK & ACCOUNT STATEMENT", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 2. STORE CONTACT & CUSTOMER METADATA
	curY := bannerH + 12.0
	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 8.5)

	if storeAddress != "" {
		pdf.SetXY(0, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 13}, storeAddress, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
		curY += 13.0
	}
	if storePhone != "" {
		pdf.SetXY(0, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 13}, storePhone, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
		curY += 13.0
	}

	// Customer Info Centered Line
	custLine := fmt.Sprintf("Customer: %s (%s) | Statement Date: %s", custName, custPhone, time.Now().Format("02-Jan-2006 15:04"))
	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 9)
	pdf.SetXY(0, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, custLine, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	curY += 20.0

	// 3. THREE FINANCIAL SUMMARY CARDS
	cardGap := 10.0
	cardW := (contentWidth - (cardGap * 2)) / 3.0
	cardH := 46.0

	// Card 1: Total Udhar Given (Debit)
	x1 := contentLeft
	pdf.SetFillColor(254, 242, 242) // #fef2f2 light red
	pdf.SetStrokeColor(254, 202, 202) // #fecaca
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(x1, curY, cardW, cardH, "FD")

	pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x1+8, curY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 12}, "TOTAL UDHAR GIVEN (DR)")
	_ = pdf.SetFont("noto-bold", "", 12)
	pdf.SetXY(x1+8, curY+20)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 18}, fmt.Sprintf("Rs. %s", gohumanize.FormatIndian(int64(totalDebit))))

	// Card 2: Total Jama Repaid (Credit)
	x2 := x1 + cardW + cardGap
	pdf.SetFillColor(240, 253, 244) // #f0fdf4 light green
	pdf.SetStrokeColor(187, 247, 208) // #bbf7d0
	pdf.RectFromUpperLeftWithStyle(x2, curY, cardW, cardH, "FD")

	pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x2+8, curY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 12}, "TOTAL REPAID / JAMA (CR)")
	_ = pdf.SetFont("noto-bold", "", 12)
	pdf.SetXY(x2+8, curY+20)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 18}, fmt.Sprintf("Rs. %s", gohumanize.FormatIndian(int64(totalCredit))))

	// Card 3: Net Outstanding Balance
	x3 := x2 + cardW + cardGap
	if currentDue > 0 {
		pdf.SetFillColor(254, 242, 242)
		pdf.SetStrokeColor(cRed[0], cRed[1], cRed[2])
		pdf.RectFromUpperLeftWithStyle(x3, curY, cardW, cardH, "FD")
		pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
	} else {
		pdf.SetFillColor(240, 253, 244)
		pdf.SetStrokeColor(cGreen[0], cGreen[1], cGreen[2])
		pdf.RectFromUpperLeftWithStyle(x3, curY, cardW, cardH, "FD")
		pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
	}
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(x3+8, curY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 12}, "CURRENT OUTSTANDING DUE")
	_ = pdf.SetFont("noto-bold", "", 13)
	pdf.SetXY(x3+8, curY+20)
	dueLabel := fmt.Sprintf("Rs. %s", gohumanize.FormatIndian(int64(currentDue)))
	if currentDue <= 0 {
		dueLabel += " (SETTLED)"
	} else {
		dueLabel += " (PENDING)"
	}
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 18}, dueLabel)

	curY += cardH + 12.0

	// 4. UPI REPAYMENT BANNER (If customer has dues and merchant has UPI)
	if storeUPI != "" && currentDue > 0 {
		upiH := 50.0
		pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
		pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
		pdf.SetLineWidth(0.6)
		pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, upiH, "FD")

		// Generate QR
		upiPayload := fmt.Sprintf("upi://pay?pa=%s&pn=%s&am=%.2f&cu=INR", storeUPI, storeName, currentDue)
		qrBytes, qrErr := qrcode.Encode(upiPayload, qrcode.Medium, 128)
		if qrErr == nil {
			qrImg, imgErr := gopdf.ImageHolderByBytes(qrBytes)
			if imgErr == nil {
				_ = pdf.ImageByHolder(qrImg, contentLeft+10, curY+4, &gopdf.Rect{W: 42, H: 42})
			}
		}

		textX := contentLeft + 60.0
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 9.5)
		pdf.SetXY(textX, curY+8)
		_ = pdf.Cell(&gopdf.Rect{W: 300, H: 14}, "Scan & Pay via UPI (PhonePe / Google Pay / Paytm)")

		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8.5)
		pdf.SetXY(textX, curY+22)
		_ = pdf.Cell(&gopdf.Rect{W: 300, H: 12}, "Merchant UPI VPA: "+storeUPI)

		pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(textX, curY+34)
		_ = pdf.Cell(&gopdf.Rect{W: 300, H: 12}, fmt.Sprintf("Exact Outstanding Dues: ₹%.2f (Instant Settlement)", currentDue))

		curY += upiH + 12.0
	}

	// 5. LEDGER TRANSACTIONS TABLE
	tblH := 22.0
	pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

	w1 := 26.0  // #
	w2 := 95.0  // Date & Time
	w3 := 168.0 // Particulars / Notes
	w4 := 80.0  // Udhar Given (Dr)
	w5 := 80.0  // Repaid Jama (Cr)
	w6 := 90.0  // Balance After

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: tblH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: tblH}, "Date & Time", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: tblH}, "Particulars / Notes", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: tblH}, "Udhar Given (Dr)", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: tblH}, "Repaid (Cr)", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4+w5-4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: tblH}, "Balance (₹)", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

	curY += tblH
	rowH := 20.0

	for i, tx := range transactions {
		if curY > 770 {
			pdf.AddPage()
			curY = 30.0
		}

		if i%2 == 1 {
			pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
			pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, rowH, "F")
		}

		pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
		pdf.SetLineWidth(0.4)
		pdf.Line(contentLeft, curY+rowH, contentRight, curY+rowH)

		// #
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		pdf.SetXY(contentLeft, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: rowH}, fmt.Sprintf("%d", i+1), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Date
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		dateStr := tx.CreatedAt.Format("02-01-2006 15:04")
		if tx.CreatedAt.IsZero() {
			dateStr = "-"
		}
		pdf.SetXY(contentLeft+w1+4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: rowH}, dateStr, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Particulars / Notes
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		notes := strings.TrimSpace(tx.Notes)
		if tx.BillNumber != "" {
			notes = "Bill: #" + tx.BillNumber + " " + notes
		}
		if notes == "" {
			if tx.Type == model.KhataTxTypeGiveCredit || tx.Type == "debit" || tx.Type == "DEBIT" {
				notes = "Udhar Purchase"
			} else {
				notes = "Cash / UPI Repayment"
			}
		}
		notes = TruncateRunes(notes, 32)
		pdf.SetXY(contentLeft+w1+w2+4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w3 - 8, H: rowH}, notes, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Udhar Given (Dr)
		isDebit := tx.Type == model.KhataTxTypeGiveCredit || tx.Type == "debit" || tx.Type == "DEBIT"
		if isDebit {
			pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
			_ = pdf.SetFont("noto-bold", "", 8.5)
			pdf.SetXY(contentLeft+w1+w2+w3, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, fmt.Sprintf("₹%.2f", tx.Amount), gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
		} else {
			pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
			_ = pdf.SetFont("noto-reg", "", 8.5)
			pdf.SetXY(contentLeft+w1+w2+w3, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, "-", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
		}

		// Repaid Jama (Cr)
		if !isDebit {
			pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
			_ = pdf.SetFont("noto-bold", "", 8.5)
			pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, fmt.Sprintf("₹%.2f", tx.Amount), gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
		} else {
			pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
			_ = pdf.SetFont("noto-reg", "", 8.5)
			pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, "-", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})
		}

		// Balance After
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2+w3+w4+w5-4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: rowH}, fmt.Sprintf("₹%.2f", tx.BalanceAfter), gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

		curY += rowH
	}

	// 6. FOOTER NOTE
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 7.8)
	pdf.SetXY(contentLeft, footerY+8)
	footerMsg := "Notes: Please verify all entries and clear outstanding dues on time. Generated automatically by ShopSilo Retail OS."
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 12}, footerMsg, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	return pdf.GetBytesPdfReturnErr()
}
