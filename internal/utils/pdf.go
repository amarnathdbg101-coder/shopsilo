// Package utils provides helper utilities.
package utils

import (
	"bytes"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// GenerateWholesaleReorderPDF generates a ready-to-print PDF purchase order sheet for wholesale suppliers.
func GenerateWholesaleReorderPDF(shop *model.Shop, items []*dto.LowStockProduct) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// 1. Header Banner
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(33, 37, 41)
	shopName := "Retail Store"
	if shop != nil && shop.Name != "" {
		shopName = shop.Name
	}
	pdf.CellFormat(0, 10, shopName, "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(108, 117, 125)
	if shop != nil {
		if shop.Address != "" {
			pdf.CellFormat(0, 5, fmt.Sprintf("Address: %s, %s %s", shop.Address, shop.City, shop.Pincode), "", 1, "L", false, 0, "")
		}
		contact := shop.Phone
		if shop.WhatsAppNumber != "" {
			contact = fmt.Sprintf("%s (WhatsApp: %s)", contact, shop.WhatsAppNumber)
		}
		if contact != "" {
			pdf.CellFormat(0, 5, fmt.Sprintf("Phone: %s", contact), "", 1, "L", false, 0, "")
		}
	}
	pdf.CellFormat(0, 5, fmt.Sprintf("Date Generated: %s", time.Now().Format("02 Jan 2006, 03:04 PM")), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Horizontal Rule
	pdf.SetDrawColor(220, 224, 230)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(6)

	// Document Title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(13, 110, 253)
	pdf.CellFormat(0, 8, "WHOLESALE RE-ORDER / PURCHASE SHEET", "", 1, "C", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(108, 117, 125)
	pdf.CellFormat(0, 5, fmt.Sprintf("Total Items Requiring Restock: %d", len(items)), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Table Header
	pdf.SetFillColor(241, 243, 245)
	pdf.SetTextColor(33, 37, 41)
	pdf.SetDrawColor(206, 212, 218)
	pdf.SetLineWidth(0.3)
	pdf.SetFont("Arial", "B", 9)

	colWidths := []float64{10, 60, 30, 25, 25, 30}
	headers := []string{"#", "Product Name", "SKU", "In-Stock", "Re-Order Qty", "Supplier / Notes"}

	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table Body
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(40, 40, 40)

	if len(items) == 0 {
		pdf.CellFormat(180, 10, "All items have sufficient stock. No re-order needed.", "1", 1, "C", false, 0, "")
	} else {
		for idx, item := range items {
			name := item.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			pdf.CellFormat(colWidths[0], 7, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
			pdf.CellFormat(colWidths[1], 7, fmt.Sprintf(" %s", name), "1", 0, "L", false, 0, "")
			pdf.CellFormat(colWidths[2], 7, fmt.Sprintf(" %s", item.SKU), "1", 0, "L", false, 0, "")

			// Red text for 0 stock, regular otherwise
			if item.CurrentStock <= 0 {
				pdf.SetTextColor(220, 53, 69)
			} else {
				pdf.SetTextColor(40, 40, 40)
			}
			pdf.CellFormat(colWidths[3], 7, fmt.Sprintf("%d", item.CurrentStock), "1", 0, "C", false, 0, "")

			pdf.SetTextColor(13, 110, 253)
			pdf.CellFormat(colWidths[4], 7, fmt.Sprintf("%d", item.SuggestedReorderQty), "1", 0, "C", false, 0, "")

			pdf.SetTextColor(40, 40, 40)
			pdf.CellFormat(colWidths[5], 7, "", "1", 1, "C", false, 0, "") // blank box for wholesale dealer note
		}
	}

	pdf.Ln(8)
	// Instructions for Wholesale Vendor
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.MultiCell(0, 4, "Notes: Please verify stock availability and provide estimated delivery time. Generated automatically by ShopMe Retail OS.", "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to output PDF: %w", err)
	}

	return buf.Bytes(), nil
}
