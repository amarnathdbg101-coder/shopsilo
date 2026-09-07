// Package utils provides helper utilities.
package utils

import (
	"bytes"
	"fmt"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// GeneratePOSReceiptPDF generates a clean digital bill/receipt in PDF format for walk-in counter sales.
func GeneratePOSReceiptPDF(bill *model.POSBill) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A5", "") // Compact A5 receipt size (148 x 210 mm)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	// 1. Store Header
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(33, 37, 41)
	storeName := "Retail Store"
	if bill.Shop != nil && bill.Shop.Name != "" {
		storeName = bill.Shop.Name
	}
	pdf.CellFormat(0, 8, storeName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(108, 117, 125)
	if bill.Shop != nil {
		if bill.Shop.Address != "" {
			pdf.CellFormat(0, 4, bill.Shop.Address, "", 1, "C", false, 0, "")
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

	// Divider line
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)
	pdf.Line(10, pdf.GetY(), 138, pdf.GetY())
	pdf.Ln(3)

	// 2. Bill Meta Information
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(13, 110, 253)
	pdf.CellFormat(0, 6, "TAX INVOICE / CASH RECEIPT", "", 1, "C", false, 0, "")
	pdf.Ln(1)

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.CellFormat(64, 4, fmt.Sprintf("Bill No: %s", bill.BillNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(64, 4, fmt.Sprintf("Date: %s", bill.CreatedAt.Format("02-Jan-2006 15:04")), "", 1, "R", false, 0, "")

	payMethod := strings.ToUpper(bill.PaymentMethod)
	pdf.CellFormat(64, 4, fmt.Sprintf("Payment: %s", payMethod), "", 0, "L", false, 0, "")
	if bill.CustomerPhone != "" {
		pdf.CellFormat(64, 4, fmt.Sprintf("Customer: %s", bill.CustomerPhone), "", 1, "R", false, 0, "")
	} else {
		pdf.CellFormat(64, 4, "Customer: Walk-in", "", 1, "R", false, 0, "")
	}
	pdf.Ln(3)

	// 3. Itemized Table
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(33, 37, 41)

	colWidths := []float64{10, 68, 15, 17, 18}
	pdf.CellFormat(colWidths[0], 6, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[1], 6, "Item Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colWidths[2], 6, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[3], 6, "Rate", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[4], 6, "Amount", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(40, 40, 40)

	for idx, item := range bill.Items {
		name := item.ProductName
		if len(name) > 35 {
			name = name[:32] + "..."
		}
		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, fmt.Sprintf(" %s", name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf("%.2f ", item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, fmt.Sprintf("%.2f ", item.TotalPrice), "1", 1, "R", false, 0, "")
	}

	// 4. Totals Calculation
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(93, 5, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 5, fmt.Sprintf("Rs. %.2f ", bill.Subtotal), "", 1, "R", false, 0, "")

	if bill.DiscountAmount > 0 {
		pdf.CellFormat(93, 5, "Discount:", "", 0, "R", false, 0, "")
		pdf.CellFormat(35, 5, fmt.Sprintf("- Rs. %.2f ", bill.DiscountAmount), "", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(13, 110, 253)
	pdf.CellFormat(93, 7, "Grand Total:", "T", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, fmt.Sprintf("Rs. %.2f ", bill.TotalAmount), "T", 1, "R", false, 0, "")

	// 5. Footer & Thank You
	pdf.Ln(6)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(0, 4, "Thank you for shopping with us! Visit again.", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, "Paperless Digital Bill by ShopMe Retail OS", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate receipt PDF: %w", err)
	}

	return buf.Bytes(), nil
}
