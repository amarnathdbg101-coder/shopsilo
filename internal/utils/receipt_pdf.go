// Package utils provides helper utilities.
package utils

import (
	"bytes"
	"fmt"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// GeneratePOSReceiptPDF generates a genius, executive-grade digital tax receipt PDF.
func GeneratePOSReceiptPDF(bill *model.POSBill) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A5", "") // Executive A5 receipt size (148 x 210 mm)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	// Colors Palette
	// Primary: Dark Indigo (15, 23, 42)
	// Accent: Bright Indigo (67, 56, 202)
	// Light Background: Slate Light (248, 250, 252)
	// Text Dark: (30, 41, 59)
	// Text Muted: (100, 116, 139)

	// 1. Top Brand Banner / Header Card
	pdf.SetFillColor(15, 23, 42) // Dark Slate Header
	pdf.Rect(0, 0, 148, 22, "F")

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(255, 255, 255)
	storeName := "RETAIL STORE"
	if bill.Shop != nil && bill.Shop.Name != "" {
		storeName = strings.ToUpper(bill.Shop.Name)
	}
	pdf.SetY(5)
	pdf.CellFormat(0, 6, storeName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 7)
	pdf.SetTextColor(199, 210, 254) // Indigo Light Accent
	pdf.CellFormat(0, 4, "OFFICIAL DIGITAL TAX INVOICE", "", 1, "C", false, 0, "")

	pdf.SetY(26)

	// 2. Store Info Sub-header
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	if bill.Shop != nil {
		if bill.Shop.Address != "" {
			pdf.CellFormat(0, 4, fmt.Sprintf("%s, %s %s", bill.Shop.Address, bill.Shop.City, bill.Shop.Pincode), "", 1, "C", false, 0, "")
		}
		contact := bill.Shop.Phone
		if bill.Shop.WhatsAppNumber != "" {
			contact = fmt.Sprintf("Phone / WhatsApp: %s", bill.Shop.WhatsAppNumber)
		}
		if contact != "" {
			pdf.CellFormat(0, 4, contact, "", 1, "C", false, 0, "")
		}
	}
	pdf.Ln(2)

	// Subtle Divider
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())
	pdf.Ln(3)

	// 3. Bill Metadata Summary Box (Two Column Layout)
	metaY := pdf.GetY()
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.RoundedRect(10, metaY, 128, 16, 2, "1234", "FD")

	pdf.SetY(metaY + 2)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(64, 4, fmt.Sprintf(" Invoice No: %s", bill.BillNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(60, 4, fmt.Sprintf("Date: %s ", bill.CreatedAt.Format("02-Jan-2006 15:04")), "", 1, "R", false, 0, "")

	payMethod := strings.ToUpper(bill.PaymentMethod)
	customerStr := "Walk-in Shopper"
	if bill.CustomerPhone != "" {
		customerStr = maskPhone(bill.CustomerPhone)
	}

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(64, 4, fmt.Sprintf(" Payment Mode: %s", payMethod), "", 0, "L", false, 0, "")
	pdf.CellFormat(60, 4, fmt.Sprintf("Customer: %s ", customerStr), "", 1, "R", false, 0, "")

	pdf.Ln(6)

	// 4. Itemized Table
	pdf.SetFillColor(67, 56, 202) // Indigo Header
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(255, 255, 255)

	colWidths := []float64{10, 64, 16, 18, 20}
	pdf.CellFormat(colWidths[0], 6, "#", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[1], 6, "Item Description", "", 0, "L", true, 0, "")
	pdf.CellFormat(colWidths[2], 6, "Qty", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[3], 6, "Rate", "", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[4], 6, "Amount ", "", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 8)

	// Zebra Striped Item Rows
	for idx, item := range bill.Items {
		if idx%2 == 0 {
			pdf.SetFillColor(255, 255, 255)
		} else {
			pdf.SetFillColor(248, 250, 252) // Light Gray/Blue stripe
		}

		name := item.ProductName
		if len(name) > 33 {
			name = name[:30] + "..."
		}

		pdf.SetTextColor(30, 41, 59)
		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "", 0, "C", true, 0, "")
		pdf.CellFormat(colWidths[1], 6, fmt.Sprintf(" %s", name), "", 0, "L", true, 0, "")
		pdf.CellFormat(colWidths[2], 6, fmt.Sprintf("%d", item.Quantity), "", 0, "C", true, 0, "")
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf("%.2f ", item.UnitPrice), "", 0, "R", true, 0, "")

		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(colWidths[4], 6, fmt.Sprintf("%.2f ", item.TotalPrice), "", 1, "R", true, 0, "")
		pdf.SetFont("Arial", "", 8)
	}

	pdf.SetDrawColor(226, 232, 240)
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())

	// 5. Totals & Payment Summary Section
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(90, 5, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(38, 5, fmt.Sprintf("Rs. %.2f ", bill.Subtotal), "", 1, "R", false, 0, "")

	if bill.DiscountAmount > 0 {
		pdf.SetTextColor(16, 185, 129) // Emerald Green for discount
		pdf.CellFormat(90, 5, "Discount Savings:", "", 0, "R", false, 0, "")
		pdf.CellFormat(38, 5, fmt.Sprintf("- Rs. %.2f ", bill.DiscountAmount), "", 1, "R", false, 0, "")
	}

	pdf.Ln(1)
	// Grand Total Callout Box
	totY := pdf.GetY()
	pdf.SetFillColor(238, 242, 255) // Light Indigo Tint
	pdf.SetDrawColor(199, 210, 254)
	pdf.RoundedRect(70, totY, 58, 9, 2, "1234", "FD")

	pdf.SetY(totY + 1)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(67, 56, 202) // Deep Indigo Text
	pdf.CellFormat(90, 7, "Grand Total:", "", 0, "R", false, 0, "")
	pdf.CellFormat(36, 7, fmt.Sprintf("Rs. %.2f ", bill.TotalAmount), "", 1, "R", false, 0, "")

	// 6. Split Payment Breakdown (if applicable)
	if bill.PaymentMethod == "split" || (bill.CashAmount > 0 || bill.OnlineAmount > 0 || bill.KhataAmount > 0) {
		pdf.Ln(2)
		pdf.SetFont("Arial", "", 8)
		if bill.CashAmount > 0 {
			pdf.SetTextColor(71, 85, 105)
			pdf.CellFormat(90, 4, "Paid in Cash:", "", 0, "R", false, 0, "")
			pdf.CellFormat(38, 4, fmt.Sprintf("Rs. %.2f ", bill.CashAmount), "", 1, "R", false, 0, "")
		}
		if bill.OnlineAmount > 0 {
			pdf.SetTextColor(71, 85, 105)
			pdf.CellFormat(90, 4, "Paid via UPI / Online:", "", 0, "R", false, 0, "")
			pdf.CellFormat(38, 4, fmt.Sprintf("Rs. %.2f ", bill.OnlineAmount), "", 1, "R", false, 0, "")
		}
		if bill.KhataAmount > 0 {
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(220, 38, 38) // Warning Crimson for credit due
			pdf.CellFormat(90, 4, "Added to Khata (Udhar Due):", "", 0, "R", false, 0, "")
			pdf.CellFormat(38, 4, fmt.Sprintf("Rs. %.2f ", bill.KhataAmount), "", 1, "R", false, 0, "")
		}
	}

	// 7. Dynamic QR Code Verification
	qrBytes, err := GenerateQRCodePNG(fmt.Sprintf("INVOICE:%s|TOTAL:%.2f|SHOP:%s", bill.BillNumber, bill.TotalAmount, storeName), 120)
	if err == nil && len(qrBytes) > 0 {
		qrHolder := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("qrcode", qrHolder, bytes.NewReader(qrBytes))

		// Render QR Code on the bottom left
		qrY := pdf.GetY()
		if qrY < 160 {
			pdf.ImageOptions("qrcode", 12, qrY, 20, 20, false, qrHolder, 0, "")
		}
	}

	// 8. Footer & Thank You
	pdf.SetY(192)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(67, 56, 202)
	pdf.CellFormat(0, 4, "Thank you for shopping with us! Visit again.", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "I", 7)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 3, "Verified Paperless Invoice by ShopMe Retail OS", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate genius receipt PDF: %w", err)
	}

	return buf.Bytes(), nil
}
