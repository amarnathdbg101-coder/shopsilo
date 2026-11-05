// Package utils provides high-performance native PDF generators for ShopSilo.
package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"shopMe/internal/handler/model"
)

// GenerateNativeKhataStatementPDF produces an ultra-fast, executive-grade customer Khata Passbook PDF
// in pure Go with zero browser dependencies and sub-30ms performance.
func GenerateNativeKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	pdf := gopdf.GoPdf{}

	// Standard A4 Size: 595 x 842 points
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := setupInvoiceFonts(&pdf); err != nil {
		return nil, err
	}

	// Palettes
	cNavy := []uint8{30, 27, 75}        // #1e1b4b
	cDark := []uint8{15, 23, 42}        // #0f172a
	cMuted := []uint8{100, 116, 139}    // #64748b
	cIndigo := []uint8{67, 56, 202}     // #4338ca
	cLightInd := []uint8{224, 231, 255} // #e0e7ff
	cAltRow := []uint8{248, 250, 252}   // #f8fafc
	cBorder := []uint8{226, 232, 240}   // #e2e8f0
	cGreen := []uint8{22, 163, 74}      // #16a34a
	cRed := []uint8{220, 38, 38}        // #dc2626

	contentLeft := 30.0
	contentWidth := 535.0
	contentRight := contentLeft + contentWidth

	// 1. Store Header
	storeName := "ShopSilo Retail Store"
	storeAddress := "Market Yard, Main Road"
	storeCity := ""
	storePhone := ""
	storeUPI := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			storeName = strings.TrimSpace(shop.Name)
		}
		if strings.TrimSpace(shop.Address) != "" {
			storeAddress = strings.TrimSpace(shop.Address)
		}
		if strings.TrimSpace(shop.City) != "" {
			storeCity = shop.City
			if shop.Pincode != "" {
				storeCity += " - " + shop.Pincode
			}
		}
		if shop.WhatsAppNumber != "" {
			storePhone = "WhatsApp: " + shop.WhatsAppNumber
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
		if tx.Type == "GIVE_CREDIT" || tx.Type == "debit" || tx.Type == "DEBIT" {
			totalDebit += tx.Amount
		} else {
			totalCredit += tx.Amount
		}
	}

	// Store Name
	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 16)
	pdf.SetXY(contentLeft, 25)
	_ = pdf.Cell(&gopdf.Rect{W: 340, H: 20}, storeName)

	// Address & Phone
	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(contentLeft, 47)
	if storeCity != "" {
		_ = pdf.Cell(&gopdf.Rect{W: 340, H: 14}, storeAddress+", "+storeCity)
	} else {
		_ = pdf.Cell(&gopdf.Rect{W: 340, H: 14}, storeAddress)
	}

	if storePhone != "" {
		pdf.SetXY(contentLeft, 61)
		_ = pdf.Cell(&gopdf.Rect{W: 340, H: 14}, storePhone)
	}

	// Right Badge: "KHATA PASSBOOK / खाता बही"
	badgeW := 150.0
	badgeX := contentRight - badgeW
	pdf.SetFillColor(cLightInd[0], cLightInd[1], cLightInd[2])
	pdf.RectFromUpperLeftWithStyle(badgeX, 25, badgeW, 20, "F")

	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 9)
	pdf.SetXY(badgeX, 25)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 20}, "📒 KHATA PASSBOOK / खाता बही", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(badgeX, 48)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 14}, "Statement Date: "+time.Now().Format("02-Jan-2006"), gopdf.CellOption{
		Align: gopdf.Right | gopdf.Middle,
	})

	// Top Divider
	pdf.SetStrokeColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.SetLineWidth(1.2)
	pdf.Line(contentLeft, 82, contentRight, 82)

	// 2. Customer Metadata Bar
	custBarY := 90.0
	custBarH := 24.0
	pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, custBarY, contentWidth, custBarH, "FD")

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9.5)
	pdf.SetXY(contentLeft+10, custBarY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 100, H: custBarH}, "Account Holder (ग्राहक):", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 9.5)
	pdf.SetXY(contentLeft+115, custBarY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: custBarH}, fmt.Sprintf("%s (%s)", custName, custPhone), gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	// 3. 3-Card Financial Summary (Total Debit, Total Credit, Current Due)
	cardY := 122.0
	cardW := (contentWidth - 20) / 3.0
	cardH := 46.0

	// Card 1: Total Given (उधार दिया)
	pdf.SetFillColor(254, 242, 242) // light red
	pdf.SetStrokeColor(254, 202, 202)
	pdf.RectFromUpperLeftWithStyle(contentLeft, cardY, cardW, cardH, "FD")

	pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(contentLeft+8, cardY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 10}, "TOTAL CREDIT TAKEN (उधार लिया)")

	_ = pdf.SetFont("noto-bold", "", 12)
	pdf.SetXY(contentLeft+8, cardY+22)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 16}, fmt.Sprintf("₹%.2f", totalDebit))

	// Card 2: Total Repaid (जमा किया)
	card2X := contentLeft + cardW + 10
	pdf.SetFillColor(240, 253, 244) // light green
	pdf.SetStrokeColor(187, 247, 208)
	pdf.RectFromUpperLeftWithStyle(card2X, cardY, cardW, cardH, "FD")

	pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(card2X+8, cardY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 10}, "TOTAL REPAID (जमा किया)")

	_ = pdf.SetFont("noto-bold", "", 12)
	pdf.SetXY(card2X+8, cardY+22)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 16}, fmt.Sprintf("₹%.2f", totalCredit))

	// Card 3: Net Balance Due (कुल बकाया)
	card3X := card2X + cardW + 10
	pdf.SetFillColor(cLightInd[0], cLightInd[1], cLightInd[2])
	pdf.SetStrokeColor(199, 210, 254)
	pdf.RectFromUpperLeftWithStyle(card3X, cardY, cardW, cardH, "FD")

	pdf.SetTextColor(cNavy[0], cNavy[1], cNavy[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(card3X+8, cardY+6)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 10}, "NET BALANCE DUE (कुल बकाया)")

	_ = pdf.SetFont("noto-bold", "", 13)
	pdf.SetXY(card3X+8, cardY+22)
	_ = pdf.Cell(&gopdf.Rect{W: cardW - 16, H: 16}, fmt.Sprintf("₹%.2f", currentDue))

	// 4. Transactions Table Header
	tblHeaderY := 178.0
	tblHeaderH := 20.0
	pdf.SetFillColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, tblHeaderY, contentWidth, tblHeaderH, "F")

	col1W := 30.0  // #
	col2W := 95.0  // Date
	col3W := 220.0 // Particulars / Notes
	col4W := 95.0  // Given / Debit
	col5W := 95.0  // Received / Credit

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, tblHeaderY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: col1W, H: tblHeaderH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	pdf.SetXY(contentLeft+col1W+5, tblHeaderY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: col2W, H: tblHeaderH}, "Date & Time", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	pdf.SetXY(contentLeft+col1W+col2W+5, tblHeaderY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: col3W, H: tblHeaderH}, "Particulars / Notes (विवरण)", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

	pdf.SetXY(contentLeft+col1W+col2W+col3W, tblHeaderY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: col4W - 5, H: tblHeaderH}, "Udhar Given (उधार)", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

	pdf.SetXY(contentLeft+col1W+col2W+col3W+col4W, tblHeaderY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: col5W - 8, H: tblHeaderH}, "Repaid (जमा)", gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

	// Render Rows
	curY := tblHeaderY + tblHeaderH
	rowH := 18.0

	if len(transactions) == 0 {
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft, curY+15)
		_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 20}, "No transaction records found in this statement.", gopdf.CellOption{Align: gopdf.Center})
		curY += 40
	} else {
		for i, tx := range transactions {
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
			_ = pdf.CellWithOption(&gopdf.Rect{W: col1W, H: rowH}, fmt.Sprintf("%d", i+1), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

			// Date
			pdf.SetXY(contentLeft+col1W+5, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: col2W, H: rowH}, tx.CreatedAt.Format("02-01-2006 15:04"), gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

			// Particulars
			desc := strings.TrimSpace(tx.Notes)
			if desc == "" {
				desc = "General Khata Entry"
			}
			if tx.BillNumber != "" {
				desc = fmt.Sprintf("%s (Bill: #%s)", desc, tx.BillNumber)
			}
			pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
			_ = pdf.SetFont("noto-bold", "", 8)
			pdf.SetXY(contentLeft+col1W+col2W+5, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: col3W - 10, H: rowH}, desc, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

			// Debit / Given
			debitStr := "-"
			creditStr := "-"
			if tx.Type == "GIVE_CREDIT" || tx.Type == "debit" || tx.Type == "DEBIT" {
				debitStr = fmt.Sprintf("₹%.2f", tx.Amount)
				pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
			} else {
				creditStr = fmt.Sprintf("₹%.2f", tx.Amount)
				pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
			}

			_ = pdf.SetFont("noto-bold", "", 8.5)
			pdf.SetXY(contentLeft+col1W+col2W+col3W, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: col4W - 5, H: rowH}, debitStr, gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

			pdf.SetXY(contentLeft+col1W+col2W+col3W+col4W, curY)
			_ = pdf.CellWithOption(&gopdf.Rect{W: col5W - 8, H: rowH}, creditStr, gopdf.CellOption{Align: gopdf.Right | gopdf.Middle})

			curY += rowH
		}
	}

	// 5. Dynamic UPI Pay QR Code for instant dues clearance
	if storeUPI != "" && currentDue > 0 {
		curY += 15
		upiPayload := fmt.Sprintf("upi://pay?pa=%s&pn=%s&am=%.2f&cu=INR", storeUPI, storeName, currentDue)
		if qrBytes, err := GenerateQRCodePNG(upiPayload, 140); err == nil {
			if imgHolder, err := gopdf.ImageHolderByBytes(qrBytes); err == nil {
				// QR Card background
				pdf.SetFillColor(cAltRow[0], cAltRow[1], cAltRow[2])
				pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
				pdf.SetLineWidth(0.6)
				pdf.RectFromUpperLeftWithStyle(contentLeft, curY, 260, 60, "FD")

				_ = pdf.ImageByHolder(imgHolder, contentLeft+8, curY+5, &gopdf.Rect{W: 50, H: 50})

				pdf.SetTextColor(cNavy[0], cNavy[1], cNavy[2])
				_ = pdf.SetFont("noto-bold", "", 9)
				pdf.SetXY(contentLeft+68, curY+12)
				_ = pdf.Cell(&gopdf.Rect{W: 180, H: 12}, "Scan & Pay via UPI (PhonePe / GPay)")

				pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
				_ = pdf.SetFont("noto-reg", "", 8)
				pdf.SetXY(contentLeft+68, curY+26)
				_ = pdf.Cell(&gopdf.Rect{W: 180, H: 10}, "UPI ID: "+storeUPI)

				pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
				_ = pdf.SetFont("noto-bold", "", 8)
				pdf.SetXY(contentLeft+68, curY+38)
				_ = pdf.Cell(&gopdf.Rect{W: 180, H: 10}, "Instant Automated Balance Settlement")
			}
		}
	}

	// 6. Footer Note
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(contentLeft, footerY+5)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 10}, "कृपया समय पर बकाया चुकाएं • Please clear outstanding dues on time.", gopdf.CellOption{Align: gopdf.Center})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 7.5)
	pdf.SetXY(contentLeft, footerY+17)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 10}, "Official Khata Passbook. Powered by ShopSilo Digital Retail OS.", gopdf.CellOption{Align: gopdf.Center})

	return pdf.GetBytesPdfReturnErr()
}

