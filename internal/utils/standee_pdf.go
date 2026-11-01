package utils

import (
	"bytes"
	"fmt"
	"strings"

	"shopMe/internal/handler/model"

	"github.com/jung-kurt/gofpdf"
)

// GenerateStoreStandeePDF generates a professional, high-resolution printable A5 counter standee PDF for merchants.
// The standee can be placed on acrylic counter stands for customers to scan and jump directly into the live catalog.
func GenerateStoreStandeePDF(shop *model.Shop) ([]byte, error) {
	// A5 Dimensions: 148 mm width x 210 mm height
	pdf := gofpdf.New("P", "mm", "A5", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	// 1. Top Header Banner (ShopSilo Indigo/Navy Brand)
	pdf.SetFillColor(30, 41, 59) // Slate 800
	pdf.Rect(0, 0, 148, 26, "F")

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(0, 6)
	pdf.CellFormat(148, 7, "SHOPSILO LOCAL STOREFRONT", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(203, 213, 225) // Slate 300
	pdf.CellFormat(148, 5, "OFFICIAL VERIFIED PARTNER MERCHANT", "", 1, "C", false, 0, "")

	// 2. Store Name & Details
	storeName := strings.ToUpper(strings.TrimSpace(shop.Name))
	if storeName == "" {
		storeName = "LOCAL RETAIL STORE"
	}

	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(15, 23, 42) // Slate 900
	pdf.SetXY(10, 32)
	pdf.CellFormat(128, 8, storeName, "", 1, "C", false, 0, "")

	categoryText := strings.TrimSpace(shop.Category)
	if categoryText == "" {
		categoryText = "General Retail Store"
	}
	if shop.City != "" {
		categoryText += " • " + strings.TrimSpace(shop.City)
	}

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 116, 139) // Slate 500
	pdf.CellFormat(128, 5, categoryText, "", 1, "C", false, 0, "")

	// 3. Counter QR Code Generation (Catalog Deep Link with UTM Tracking)
	targetURL := fmt.Sprintf("https://shopsilo.in/s/%s?utm_source=counter_qr&utm_medium=standee&utm_campaign=store_visit", shop.Slug)
	qrBytes, err := GenerateQRCodePNG(targetURL, 400)
	if err == nil && len(qrBytes) > 0 {
		qrHolder := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("standee_qr", qrHolder, bytes.NewReader(qrBytes))

		// Draw QR background card
		pdf.SetFillColor(248, 250, 252) // Slate 50
		pdf.SetDrawColor(226, 232, 240) // Slate 200
		pdf.RoundedRect(29, 48, 90, 96, 4, "1234", "FD")

		// Embed QR Code Image in center (70 mm x 70 mm)
		pdf.ImageOptions("standee_qr", 39, 52, 70, 70, false, qrHolder, 0, "")

		// Call to Action inside QR Card
		pdf.SetFont("Arial", "B", 10)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetXY(29, 124)
		pdf.CellFormat(90, 5, "SCAN WITH ANY CAMERA", "", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 7.5)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetXY(29, 129)
		pdf.CellFormat(90, 4, "Browse Live Catalog & Get Exclusive In-Store Offers", "", 1, "C", false, 0, "")
	}

	// 4. Payment / UPI Information (If merchant has UPI ID)
	currY := 148.0
	if strings.TrimSpace(shop.UPIID) != "" {
		pdf.SetFillColor(241, 245, 249) // Slate 100
		pdf.SetDrawColor(203, 213, 225)
		pdf.RoundedRect(14, currY, 120, 22, 3, "1234", "FD")

		pdf.SetFont("Arial", "B", 8)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetXY(14, currY+2)
		pdf.CellFormat(120, 4, "ACCEPTED DIRECT UPI PAYMENTS", "", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "B", 9)
		pdf.SetTextColor(30, 41, 59)
		pdf.SetXY(14, currY+7)
		pdf.CellFormat(120, 5, fmt.Sprintf("UPI ID: %s", strings.TrimSpace(shop.UPIID)), "", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(100, 116, 139)
		pdf.SetXY(14, currY+13)
		pdf.CellFormat(120, 4, "Google Pay  |  PhonePe  |  Paytm  |  BHIM UPI", "", 1, "C", false, 0, "")

		currY += 25
	} else {
		currY += 5
	}

	// 5. Store Address & Contact Footer
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	if strings.TrimSpace(shop.Address) != "" {
		pdf.SetXY(10, currY)
		pdf.CellFormat(128, 4, fmt.Sprintf("Store Address: %s", strings.TrimSpace(shop.Address)), "", 1, "C", false, 0, "")
		currY += 5
	}

	phoneText := shop.WhatsAppNumber
	if phoneText == "" {
		phoneText = shop.Phone
	}
	if phoneText != "" {
		pdf.SetXY(10, currY)
		pdf.CellFormat(128, 4, fmt.Sprintf("WhatsApp / Helpline: +91 %s", phoneText), "", 1, "C", false, 0, "")
		currY += 6
	}

	// 6. Bottom Brand Strip
	pdf.SetFillColor(30, 41, 59)
	pdf.Rect(0, 202, 148, 8, "F")
	pdf.SetFont("Arial", "I", 7)
	pdf.SetTextColor(226, 232, 240)
	pdf.SetXY(0, 203)
	pdf.CellFormat(148, 6, "Powered by ShopSilo O2O Platform  •  shopsilo.in", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to export standee PDF: %w", err)
	}

	return buf.Bytes(), nil
}

