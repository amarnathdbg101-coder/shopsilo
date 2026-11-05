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

	pageW := 595.28
	margin := 28.0
	contentWidth := pageW - (margin * 2) // 539.28
	contentLeft := margin
	contentRight := contentLeft + contentWidth

	// Executive Color Palette
	cTopBanner := []uint8{15, 23, 42}   // #0f172a (Dark Navy Executive Bar)
	cHeaderSub := []uint8{148, 163, 184} // #94a3b8 (Slate Muted)
	cDark := []uint8{15, 23, 42}        // #0f172a (Primary text)
	cMuted := []uint8{71, 85, 105}      // #475569 (Secondary text)
	cTblHeader := []uint8{49, 46, 129}  // #312e81 (Deep Indigo Table Header)
	cAltRow := []uint8{248, 250, 252}   // #f8fafc (Alternate row)
	cBorder := []uint8{226, 232, 240}   // #e2e8f0 (Grid borders)
	cCardBg := []uint8{241, 245, 249}   // #f1f5f9 (Pill background)
	cCardBorder := []uint8{203, 213, 225} // #cbd5e1 (Pill border)
	cBlueAccent := []uint8{30, 58, 138} // #1e3a8a (Pill text)
	cRed := []uint8{220, 38, 38}        // #dc2626 (Stock alert)

	storeName := "ShopSilo Partner Store"
	storeAddress := ""
	storePhone := ""

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
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, strings.ToUpper(storeName)+" - MANDI PROCUREMENT & REORDER SHEET", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 2. STORE CONTACT & DATE SUB-HEADER
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
	dateStr := "Date Generated: " + time.Now().Format("02-Jan-2006 15:04")
	pdf.SetXY(0, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 13}, dateStr, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	curY += 18.0

	// 3. SUMMARY CALLOUT PILL
	pillH := 24.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, pillH, "FD")

	pdf.SetTextColor(cBlueAccent[0], cBlueAccent[1], cBlueAccent[2])
	_ = pdf.SetFont("noto-bold", "", 9.5)
	pdf.SetXY(contentLeft, curY)
	pillText := fmt.Sprintf("TOTAL ITEMS ON PROCUREMENT SHEET: %d", len(items))
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: pillH}, pillText, gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})
	curY += pillH + 12.0

	// 4. DATA TABLE HEADER
	tblH := 22.0
	pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

	w1 := 26.0  // #
	w2 := 220.0 // Item Description & Category
	w3 := 75.0  // Current
	w4 := 85.0  // Required Qty
	w5 := 85.0  // Mandi Dealer
	w6 := 48.0  // Check

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: tblH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+6, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: tblH}, "Item Description & SKU", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: tblH}, "Current Stock", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: tblH}, "Required Qty", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: tblH}, "Wholesaler", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: tblH}, "Check", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	curY += tblH
	rowH := 20.0

	for i, item := range items {
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

		// Name & SKU
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		displayName := item.Name
		if item.SKU != "" {
			displayName += " (" + item.SKU + ")"
		}
		if len(displayName) > 42 {
			displayName = displayName[:39] + "..."
		}
		pdf.SetXY(contentLeft+w1+6, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w2 - 8, H: rowH}, displayName, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Current Stock
		pdf.SetTextColor(cRed[0], cRed[1], cRed[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: rowH}, fmt.Sprintf("%d", item.CurrentStock), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Required Qty
		pdf.SetTextColor(cBlueAccent[0], cBlueAccent[1], cBlueAccent[2])
		_ = pdf.SetFont("noto-bold", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, fmt.Sprintf("%d units", item.SuggestedReorderQty), gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Wholesaler
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, "-", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Checkbox
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: rowH}, "[   ]", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		curY += rowH
	}

	// 5. FOOTER NOTE
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 7.8)
	pdf.SetXY(contentLeft, footerY+8)
	footerMsg := "Notes: Please verify stock availability and wholesale rates. Generated automatically by ShopSilo Retail OS."
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 12}, footerMsg, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	return pdf.GetBytesPdfReturnErr()
}

// GenerateNativeCustomProcurementPDF creates a custom procurement sheet matching executive corporate styling.
func GenerateNativeCustomProcurementPDF(shop *model.Shop, title string, items []dto.ProcurementPDFItem) ([]byte, error) {
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

	// Executive Color Palette
	cTopBanner := []uint8{15, 23, 42}   // #0f172a (Dark Navy Executive Bar)
	cHeaderSub := []uint8{148, 163, 184} // #94a3b8 (Slate Muted)
	cDark := []uint8{15, 23, 42}        // #0f172a (Primary text)
	cMuted := []uint8{71, 85, 105}      // #475569 (Secondary text)
	cTblHeader := []uint8{49, 46, 129}  // #312e81 (Deep Indigo Table Header)
	cAltRow := []uint8{248, 250, 252}   // #f8fafc (Alternate row)
	cBorder := []uint8{226, 232, 240}   // #e2e8f0 (Grid borders)
	cCardBg := []uint8{241, 245, 249}   // #f1f5f9 (Pill background)
	cCardBorder := []uint8{203, 213, 225} // #cbd5e1 (Pill border)
	cBlueAccent := []uint8{30, 58, 138} // #1e3a8a (Pill text)

	storeName := "ShopSilo Partner Store"
	storeAddress := ""
	storePhone := ""

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
	}

	sheetTitle := title
	if sheetTitle == "" {
		sheetTitle = storeName + " - MANDI PROCUREMENT SHEET"
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
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 14}, strings.ToUpper(sheetTitle), gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 2. STORE CONTACT & DATE SUB-HEADER
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
	dateStr := "Date Generated: " + time.Now().Format("02-Jan-2006 15:04")
	pdf.SetXY(0, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: pageW, H: 13}, dateStr, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	curY += 18.0

	// 3. SUMMARY CALLOUT PILL
	pillH := 24.0
	pdf.SetFillColor(cCardBg[0], cCardBg[1], cCardBg[2])
	pdf.SetStrokeColor(cCardBorder[0], cCardBorder[1], cCardBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, pillH, "FD")

	pdf.SetTextColor(cBlueAccent[0], cBlueAccent[1], cBlueAccent[2])
	_ = pdf.SetFont("noto-bold", "", 9.5)
	pdf.SetXY(contentLeft, curY)
	pillText := fmt.Sprintf("TOTAL ITEMS ON PROCUREMENT SHEET: %d", len(items))
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: pillH}, pillText, gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})
	curY += pillH + 12.0

	// 4. DATA TABLE HEADER
	tblH := 22.0
	pdf.SetFillColor(cTblHeader[0], cTblHeader[1], cTblHeader[2])
	pdf.RectFromUpperLeftWithStyle(contentLeft, curY, contentWidth, tblH, "F")

	w1 := 26.0  // #
	w2 := 200.0 // Item Description & Category
	w3 := 80.0  // Required Qty
	w4 := 75.0  // Est. Rate
	w5 := 110.0 // Mandi Dealer / Wholesaler
	w6 := 48.0  // Check

	pdf.SetTextColor(255, 255, 255)
	_ = pdf.SetFont("noto-bold", "", 8.5)

	pdf.SetXY(contentLeft, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w1, H: tblH}, "#", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+6, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w2, H: tblH}, "Item Description & Category", gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: tblH}, "Required Qty", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: tblH}, "Est. Rate", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: tblH}, "Mandi Dealer / Wholesaler", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
	pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: tblH}, "Check", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	curY += tblH
	rowH := 20.0

	for i, item := range items {
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

		// Item Description & Category
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		itemTitle := strings.TrimSpace(item.Name)
		if item.Category != "" {
			itemTitle += " (" + strings.TrimSpace(item.Category) + ")"
		}
		if len(itemTitle) > 40 {
			itemTitle = itemTitle[:37] + "..."
		}
		pdf.SetXY(contentLeft+w1+6, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w2 - 8, H: rowH}, itemTitle, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle})

		// Required Qty
		pdf.SetTextColor(cBlueAccent[0], cBlueAccent[1], cBlueAccent[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w3, H: rowH}, item.Qty, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Est. Rate (Sanitize any legacy corrupted characters)
		rateStr := cleanPriceString(item.EstimatedPrice)
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-bold", "", 8.5)
		pdf.SetXY(contentLeft+w1+w2+w3, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w4, H: rowH}, rateStr, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Mandi Dealer / Wholesaler
		dealer := item.Dealer
		if strings.TrimSpace(dealer) == "" {
			dealer = "-"
		}
		pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
		_ = pdf.SetFont("noto-reg", "", 8)
		pdf.SetXY(contentLeft+w1+w2+w3+w4, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w5, H: rowH}, dealer, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		// Checkbox
		pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
		_ = pdf.SetFont("noto-reg", "", 9)
		pdf.SetXY(contentLeft+w1+w2+w3+w4+w5, curY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: w6, H: rowH}, "[   ]", gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

		curY += rowH
	}

	// 5. FOOTER NOTE
	footerY := 800.0
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(0.6)
	pdf.Line(contentLeft, footerY, contentRight, footerY)

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 7.8)
	pdf.SetXY(contentLeft, footerY+8)
	footerMsg := "Notes: Please verify stock availability and wholesale rates. Generated automatically by ShopSilo Retail OS."
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth, H: 12}, footerMsg, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})

	return pdf.GetBytesPdfReturnErr()
}

// cleanPriceString sanitizes any legacy garbled currency text like â‚¹ or Â,1
func cleanPriceString(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "₹0"
	}
	// Strip out mojibake
	s = strings.ReplaceAll(s, "â‚¹", "₹")
	s = strings.ReplaceAll(s, "Â", "")
	s = strings.ReplaceAll(s, ",1", "₹")
	s = strings.ReplaceAll(s, ",1", "₹")
	s = strings.ReplaceAll(s, "", "")
	if !strings.HasPrefix(s, "₹") && !strings.HasPrefix(s, "Rs") {
		s = "₹" + s
	}
	return s
}
