// Package utils provides high-performance native PDF generators for ShopSilo.
package utils

import (
	"fmt"
	"strings"

	"github.com/signintech/gopdf"
	"shopMe/internal/handler/model"
)

// GenerateNativeStoreStandeePDF generates an ultra-crisp, printable A5 counter tabletop QR standee.
func GenerateNativeStoreStandeePDF(shop *model.Shop) ([]byte, error) {
	pdf := gopdf.GoPdf{}

	// A5 Size: 420 x 595 points
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA5})
	pdf.AddPage()

	if err := setupInvoiceFonts(&pdf); err != nil {
		return nil, err
	}

	cNavy := []uint8{30, 27, 75}
	cDark := []uint8{15, 23, 42}
	cMuted := []uint8{100, 116, 139}
	cIndigo := []uint8{67, 56, 202}
	cLightInd := []uint8{224, 231, 255}
	cBorder := []uint8{226, 232, 240}
	cGreen := []uint8{22, 163, 74}

	contentLeft := 25.0
	contentWidth := 370.0

	storeName := "ShopSilo Retail Store"
	category := "General Kirana & Retail"
	storeAddress := "Main Market Yard"
	storePhone := ""
	storeUPI := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			storeName = strings.TrimSpace(shop.Name)
		}
		if strings.TrimSpace(shop.Category) != "" {
			category = strings.TrimSpace(shop.Category)
		}
		if strings.TrimSpace(shop.Address) != "" {
			storeAddress = strings.TrimSpace(shop.Address)
		}
		if strings.TrimSpace(shop.City) != "" {
			storeAddress += ", " + shop.City
		}
		if shop.WhatsAppNumber != "" {
			storePhone = shop.WhatsAppNumber
		} else if shop.Phone != "" {
			storePhone = shop.Phone
		}
		storeUPI = shop.UPIID
	}

	// 1. Elegant Standee Border Box
	pdf.SetStrokeColor(cIndigo[0], cIndigo[1], cIndigo[2])
	pdf.SetLineWidth(2.5)
	pdf.RectFromUpperLeftWithStyle(contentLeft, 25, contentWidth, 545, "D")

	// Inner thin decorative border
	pdf.SetStrokeColor(cLightInd[0], cLightInd[1], cLightInd[2])
	pdf.SetLineWidth(1.0)
	pdf.RectFromUpperLeftWithStyle(contentLeft+5, 30, contentWidth-10, 535, "D")

	// 2. Header Branding Tag
	pdf.SetFillColor(cLightInd[0], cLightInd[1], cLightInd[2])
	badgeW := 180.0
	badgeX := contentLeft + (contentWidth-badgeW)/2.0
	pdf.RectFromUpperLeftWithStyle(badgeX, 45, badgeW, 22, "F")

	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 9)
	pdf.SetXY(badgeX, 45)
	_ = pdf.CellWithOption(&gopdf.Rect{W: badgeW, H: 22}, "✨ OFFICIAL STORE QR STANDEE", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 3. Store Name
	pdf.SetTextColor(cNavy[0], cNavy[1], cNavy[2])
	_ = pdf.SetFont("noto-bold", "", 18)
	pdf.SetXY(contentLeft+10, 80)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 24}, storeName, gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// Category Subtitle
	pdf.SetTextColor(cIndigo[0], cIndigo[1], cIndigo[2])
	_ = pdf.SetFont("noto-bold", "", 10)
	pdf.SetXY(contentLeft+10, 108)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 14}, "• "+category+" •", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// Address & Contact
	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9)
	pdf.SetXY(contentLeft+10, 126)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 14}, storeAddress, gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	if storePhone != "" {
		pdf.SetXY(contentLeft+10, 142)
		_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 14}, "Call / WhatsApp: "+storePhone, gopdf.CellOption{
			Align: gopdf.Center | gopdf.Middle,
		})
	}

	// 4. Large Standee QR Code
	qrPayload := fmt.Sprintf("https://shopsilo.in/store/%s", strings.ToLower(strings.ReplaceAll(storeName, " ", "-")))
	if shop != nil && shop.ID != "" {
		qrPayload = fmt.Sprintf("https://shopsilo.in/store/%s", shop.ID)
	}

	qrSize := 170.0
	qrX := contentLeft + (contentWidth-qrSize)/2.0
	qrY := 175.0

	// QR Container Card
	pdf.SetFillColor(255, 255, 255)
	pdf.SetStrokeColor(cBorder[0], cBorder[1], cBorder[2])
	pdf.SetLineWidth(1.5)
	pdf.RectFromUpperLeftWithStyle(qrX-10, qrY-10, qrSize+20, qrSize+20, "FD")

	if qrBytes, err := GenerateQRCodePNG(qrPayload, 260); err == nil {
		if imgHolder, err := gopdf.ImageHolderByBytes(qrBytes); err == nil {
			_ = pdf.ImageByHolder(imgHolder, qrX, qrY, &gopdf.Rect{W: qrSize, H: qrSize})
		}
	}

	// 5. Action Callout Below QR
	actY := qrY + qrSize + 22.0

	pdf.SetTextColor(cNavy[0], cNavy[1], cNavy[2])
	_ = pdf.SetFont("noto-bold", "", 13)
	pdf.SetXY(contentLeft+10, actY)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 18}, "📱 Scan QR to View Live Storefront & Offers", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 9.5)
	pdf.SetXY(contentLeft+10, actY+20)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 14}, "No app download needed • Opens directly in mobile browser", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	// 6. UPI Payment Accepted Banner
	if storeUPI != "" {
		upiY := actY + 44.0
		upiW := 280.0
		upiX := contentLeft + (contentWidth-upiW)/2.0
		pdf.SetFillColor(240, 253, 244) // light green
		pdf.SetStrokeColor(187, 247, 208)
		pdf.SetLineWidth(0.8)
		pdf.RectFromUpperLeftWithStyle(upiX, upiY, upiW, 26, "FD")

		pdf.SetTextColor(cGreen[0], cGreen[1], cGreen[2])
		_ = pdf.SetFont("noto-bold", "", 9.5)
		pdf.SetXY(upiX, upiY)
		_ = pdf.CellWithOption(&gopdf.Rect{W: upiW, H: 26}, "💳 Accepted: UPI ("+storeUPI+") / Cash", gopdf.CellOption{
			Align: gopdf.Center | gopdf.Middle,
		})
	}

	// 7. Footer Branding
	pdf.SetTextColor(cDark[0], cDark[1], cDark[2])
	_ = pdf.SetFont("noto-bold", "", 10)
	pdf.SetXY(contentLeft+10, 525)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 14}, "ShopSilo Smart Retail Operating System", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	pdf.SetTextColor(cMuted[0], cMuted[1], cMuted[2])
	_ = pdf.SetFont("noto-reg", "", 8)
	pdf.SetXY(contentLeft+10, 541)
	_ = pdf.CellWithOption(&gopdf.Rect{W: contentWidth - 20, H: 12}, "www.shopsilo.in • Empowering Local Dukan", gopdf.CellOption{
		Align: gopdf.Center | gopdf.Middle,
	})

	return pdf.GetBytesPdfReturnErr()
}

