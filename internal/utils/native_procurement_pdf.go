// Package utils provides high-performance native PDF generators for ShopSilo.
package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

// GenerateNativeWholesaleReorderPDF creates an executive A4 purchase order sheet for mandi/wholesale restock.
func GenerateNativeWholesaleReorderPDF(shop *model.Shop, items []*dto.LowStockProduct) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := setupInvoiceFonts(&pdf); err != nil {
		return nil, err
	}

	cNavy := []uint8{30, 27, 75}
	cDark := []uint8{15, 23, 42}
	cMuted := []uint8{100, 116, 139}
	cIndigo := []uint8{67, 56, 202}
	cLightInd := []uint8{224, 231, 255}
	cAltRow := []uint8{248, 250, 252}
	cBorder := []uint8{226, 232, 240}
	cRed := []uint8{220, 38, 38}

	contentLeft := 30.0
	contentWidth := 535.0
	contentRight := contentLeft + contentWidth

	storeName := "ShopSilo Retail Store"
	storeAddress := "Market Yard, Main Road"
	storeCity := ""
	storePhone := ""

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
	}

	// 1. Header
	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 16)
	pdf.SetXY(contentLeft, 25)
	_ = pdf.Cell(&gopdf.Rect{W: 320, H: 20}, storeName)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(contentLeft, 47)
	if storeCity != "" {
		_ = pdf.Cell(&gopdf.Rect{W: 320, H: 14}, storeAddress+", "+storeCity)
	} else {
		_ = pdf.Cell(&gopdf.Rect{W: 320, H: 14}, storeAddress)
	}

	if storePhone != "" {
		pdf.SetXY(contentLeft, 61)
		_ = pdf.Cell(&gopdf.Rect{W: 320, H: 14}, storePhone)
	}

	// Right Badge
	badgeW := 190.0
	badgeX := contentRight - badgeW
	pdf.SetFillColor(cLightInd[0], cLightInd[1], cLightInd[2])
	pdf.RectFromUpperLeftWithStyle(badgeX, 25, badgeW, 20, "F")

	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(badgeX, 25)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 20}, "📦 MANDI & WHOLESALE PROCUREMENT", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(badgeX, 48)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 14}, "Date: "+time.Now().Format("02-Jan-2006 15:04"), gopdf.CellOption{
		Align: gopdf.Right | gopdf.Middle,
	})

	// Divider
	pdf.SetStrokeColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.SetLineWidth(1.2)
	pdf.Line(contentLeft, 82, contentRight, 82)

	// Summary Callout Box
	boxY := 90.0
	boxH := 24.0
	pdf.SetFillColor(238, 242, 255)
	pdf.SetStrokeColor(199, 210, 254)
	pdf.SetLineWidth(0.8)
	pdf.RectFromUpperLeftWithStyle(contentLeft, boxY, contentWidth, boxH, "FD")

	pdf.SetTextColor(cNavy[0], cNavy[1], cNavy[2])
	_ = pdf.SetFont("noto-bold", "", 9.5)
	pdf.SetXY(contentLeft, boxY)
	summaryText := fmt.Sprintf("⚠️ URGENT RESTOCK REQUIRED: %d item(s) currently below critical safety threshold", len(items))
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: boxH}, summaryText, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// Table Header
	tblY := 122.0
	tblH := 20.0
	pdf.SetFillColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, tblY, contentWidth, tblH, "F")

	w1 := 30.0  // #
	w2 := 230.0 // Item Name
	w3 := 90.0  // SKU
	w4 := 55.0  // Current
	w5 := 75.0  // Suggested
	w6 := 55.0  // Checkbox

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: tblH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+5, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: tblH}, "Item Description (सामान का विवरण)", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: tblH}, "SKU / Code", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: tblH}, "Current", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: tblH}, "Reorder Qty", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: tblH}, "Received", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	// Render rows
	curY := tblY + tblH
	rowH := 18.0

	for i, item := range items {
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

		// Name
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w2 - 10, H: rowH}, item.Name, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// SKU
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		pdf.SetXY(contentLeft+w1+w2, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: rowH}, item.SKU, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Current
		pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2+w3, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, fmt.Sprintf("%d", item.CurrentStock), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Suggested
		pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
		_ = pdf.SetFont("noto-bold", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, fmt.Sprintf("%d units", item.SuggestedReorderQty), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Checkbox
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: rowH}, "[   ]", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		curY += rowH
	}

	// Footer Note & Verification
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(contentLeft, footerY+8)
	_ = pdf.Cell(&gopdf.Rect{W: 300, H: 12}, "Wholesaler Signature / Stamp: _______________________")

	pdf.SetXY(contentRight-200, footerY+8)
	_ = pdf.CellWithOption(&gopdf.Rect{W: 200, H: 12}, "Verified by Store Manager: [   ]", gopdf.CellOption{Align: gopdf.Right})

	return pdf.GetBytesPdfReturnErr()
}

// GenerateNativeCustomProcurementPDF creates a custom procurement sheet.
func GenerateNativeCustomProcurementPDF(shop *model.Shop, title string, items []dto.ProcurementPDFItem) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := setupInvoiceFonts(&pdf); err != nil {
		return nil, err
	}

	cNavy := []uint8{30, 27, 75}
	cDark := []uint8{15, 23, 42}
	cMuted := []uint8{100, 116, 139}
	cIndigo := []uint8{67, 56, 202}
	cLightInd := []uint8{224, 231, 255}
	cAltRow := []uint8{248, 250, 252}
	cBorder := []uint8{226, 232, 240}

	contentLeft := 30.0
	contentWidth := 535.0
	contentRight := contentLeft + contentWidth

	storeName := "ShopSilo Retail Store"
	if shop != nil && strings.TrimSpace(shop.Name) != "" {
		storeName = strings.TrimSpace(shop.Name)
	}

	// Header
	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 16)
	pdf.SetXY(contentLeft, 25)
	_ = pdf.Cell(&gopdf.Rect{W: 340, H: 20}, storeName)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(contentLeft, 47)
	sheetTitle := title
	if sheetTitle == "" {
		sheetTitle = "Special Order Procurement Manifest"
	}
	_ = pdf.Cell(&gopdf.Rect{W: 340, H: 14}, sheetTitle)

	// Right Badge
	badgeW := 150.0
	badgeX := contentRight - badgeW
	pdf.SetFillColor(cLightInd[0], cLightInd[1], cLightInd[2])
	pdf.RectFromUpperLeftWithStyle(badgeX, 25, badgeW, 20, "F")

	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 8.5)
	pdf.SetXY(badgeX, 25)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 20}, "📋 PROCUREMENT LIST", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(badgeX, 48)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 14}, "Date: "+time.Now().Format("02-Jan-2006"), gopdf.CellOption{
		Align: gopdf.Right | gopdf.Middle,
	})

	// Divider
	pdf.SetStrokeColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.SetLineWidth(1.2)
	pdf.Line(contentLeft, 75, contentRight, 75)

	// Table Header
	tblY := 85.0
	tblH := 20.0
	pdf.SetFillColor(cNavy[0], cNavy[1], cNavy[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, tblY, contentWidth, tblH, "F")

	w1 := 30.0  // #
	w2 := 220.0 // Item Description
	w3 := 60.0  // Qty
	w4 := 80.0  // Estimated Rate
	w5 := 90.0  // Dealer / Wholesaler
	w6 := 55.0  // Checkbox

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: tblH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+5, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: tblH}, "Item Description & Category", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: tblH}, "Qty", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: tblH}, "Est. Rate", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: tblH}, "Dealer / Mandi", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, tblY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: tblH}, "Status", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	curY := tblY + tblH
	rowH := 18.0

	for i, it := range items {
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

		// Name & Category
		desc := it.Name
		if it.Category != "" {
			desc += " (" + it.Category + ")"
		}
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w2 - 10, H: rowH}, desc, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Qty
		pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: rowH}, it.Qty, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Rate
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		pdf.SetXY(contentLeft+w1+w2+w3, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, it.EstimatedPrice, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Dealer
		pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, it.Dealer, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Checkbox
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: rowH}, "[   ]", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		curY += rowH
	}

	return pdf.GetBytesPdfReturnErr()
}

