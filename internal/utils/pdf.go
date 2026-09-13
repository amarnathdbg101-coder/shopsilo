// Package utils provides helper utilities.
package utils

import (
	"bytes"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// GenerateWholesaleReorderPDF generates an executive, ready-to-print PDF purchase order sheet for wholesale suppliers.
func GenerateWholesaleReorderPDF(shop *model.Shop, items []*dto.LowStockProduct) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	// 1. Top Brand Banner
	pdf.SetFillColor(15, 23, 42) // Slate Navy Accent Header
	pdf.Rect(0, 0, 210, 24, "F")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(255, 255, 255)
	shopName := "RETAIL STORE"
	if shop != nil && shop.Name != "" {
		shopName = strings.ToUpper(shop.Name)
	}
	pdf.SetY(6)
	pdf.CellFormat(0, 7, shopName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(199, 210, 254)
	pdf.CellFormat(0, 4, "WHOLESALE RE-ORDER & PROCUREMENT SHEET", "", 1, "C", false, 0, "")

	pdf.SetY(28)

	// 2. Store Info Sub-header
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	if shop != nil {
		if shop.Address != "" {
			pdf.CellFormat(0, 4, fmt.Sprintf("%s, %s %s", shop.Address, shop.City, shop.Pincode), "", 1, "C", false, 0, "")
		}
		contact := shop.Phone
		if shop.WhatsAppNumber != "" {
			contact = fmt.Sprintf("Phone / WhatsApp: %s", shop.WhatsAppNumber)
		}
		if contact != "" {
			pdf.CellFormat(0, 4, contact, "", 1, "C", false, 0, "")
		}
	}
	pdf.CellFormat(0, 4, fmt.Sprintf("Date Generated: %s", time.Now().Format("02-Jan-2006 15:04")), "", 1, "C", false, 0, "")
	pdf.Ln(2)

	// Horizontal Rule
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())
	pdf.Ln(4)

	// 3. Document Summary Callout
	sumY := pdf.GetY()
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.RoundedRect(12, sumY, 186, 10, 2, "1234", "FD")

	pdf.SetY(sumY + 2)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(67, 56, 202)
	pdf.CellFormat(186, 6, fmt.Sprintf("TOTAL ITEMS REQUIRING MANDI / WHOLESALE RESTOCK: %d", len(items)), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// 4. Table Header
	pdf.SetFillColor(67, 56, 202) // Deep Indigo Header
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(255, 255, 255)

	colWidths := []float64{10, 72, 32, 24, 26, 22}
	headers := []string{"#", "Product Description", "SKU / Code", "Current Stock", "Re-Order Qty", "Check [ ]"}

	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// 5. Zebra Striped Table Body
	pdf.SetFont("Arial", "", 8)

	if len(items) == 0 {
		pdf.CellFormat(186, 10, "Procurement list is empty. All store items are currently well-stocked.", "", 1, "C", false, 0, "")
	} else {
		for idx, item := range items {
			if idx%2 == 0 {
				pdf.SetFillColor(255, 255, 255)
			} else {
				pdf.SetFillColor(248, 250, 252)
			}

			name := item.Name
			if len(name) > 38 {
				name = name[:35] + "..."
			}

			skuStr := item.SKU
			if skuStr == "" {
				skuStr = "N/A"
			}

			pdf.SetTextColor(30, 41, 59)
			pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "", 0, "C", true, 0, "")
			pdf.CellFormat(colWidths[1], 6, fmt.Sprintf(" %s", name), "", 0, "L", true, 0, "")
			pdf.CellFormat(colWidths[2], 6, fmt.Sprintf(" %s", skuStr), "", 0, "C", true, 0, "")

			// Red text for 0 stock, regular otherwise
			if item.CurrentStock <= 0 {
				pdf.SetTextColor(220, 38, 38)
				pdf.SetFont("Arial", "B", 8)
			} else {
				pdf.SetTextColor(30, 41, 59)
				pdf.SetFont("Arial", "", 8)
			}
			pdf.CellFormat(colWidths[3], 6, fmt.Sprintf("%d", item.CurrentStock), "", 0, "C", true, 0, "")

			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(67, 56, 202)
			pdf.CellFormat(colWidths[4], 6, fmt.Sprintf("%d", item.SuggestedReorderQty), "", 0, "C", true, 0, "")

			pdf.SetTextColor(100, 116, 139)
			pdf.CellFormat(colWidths[5], 6, "[   ]", "", 1, "C", true, 0, "") // Checkbox for physical Mandi procurement
			pdf.SetFont("Arial", "", 8)
		}
	}

	pdf.SetDrawColor(226, 232, 240)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())

	// 6. Footer Notes
	pdf.SetY(280)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 3, "Notes: Please verify stock availability and wholesale rates. Generated automatically by ShopMe Retail OS.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to output PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCustomProcurementPDF generates an executive PDF purchase order sheet directly from screen procurement items.
func GenerateCustomProcurementPDF(shop *model.Shop, title string, items []dto.ProcurementPDFItem) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	// 1. Header Banner
	pdf.SetFillColor(15, 23, 42) // Slate Navy Accent Header
	pdf.Rect(0, 0, 210, 24, "F")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(255, 255, 255)
	shopName := "RETAIL STORE"
	if shop != nil && shop.Name != "" {
		shopName = strings.ToUpper(shop.Name)
	}
	pdf.SetY(6)
	pdf.CellFormat(0, 7, shopName, "", 1, "C", false, 0, "")

	docTitle := "MANDI KHAREED & WHOLESALE PROCUREMENT SHEET"
	if strings.TrimSpace(title) != "" {
		docTitle = strings.ToUpper(strings.TrimSpace(title))
	}

	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(199, 210, 254)
	pdf.CellFormat(0, 4, docTitle, "", 1, "C", false, 0, "")

	pdf.SetY(28)

	// 2. Sub-header
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	if shop != nil {
		if shop.Address != "" {
			pdf.CellFormat(0, 4, fmt.Sprintf("%s, %s %s", shop.Address, shop.City, shop.Pincode), "", 1, "C", false, 0, "")
		}
		contact := shop.Phone
		if shop.WhatsAppNumber != "" {
			contact = fmt.Sprintf("Phone / WhatsApp: %s", shop.WhatsAppNumber)
		}
		if contact != "" {
			pdf.CellFormat(0, 4, contact, "", 1, "C", false, 0, "")
		}
	}
	pdf.CellFormat(0, 4, fmt.Sprintf("Date Generated: %s", time.Now().Format("02-Jan-2006 15:04")), "", 1, "C", false, 0, "")
	pdf.Ln(2)

	// Horizontal Rule
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())
	pdf.Ln(4)

	// 3. Document Summary Callout
	sumY := pdf.GetY()
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.RoundedRect(12, sumY, 186, 10, 2, "1234", "FD")

	pdf.SetY(sumY + 2)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(67, 56, 202)
	pdf.CellFormat(186, 6, fmt.Sprintf("TOTAL ITEMS ON PROCUREMENT SHEET: %d", len(items)), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// 4. Table Header
	pdf.SetFillColor(67, 56, 202)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(255, 255, 255)

	colWidths := []float64{10, 92, 38, 20, 26}
	headers := []string{"#", "Item Description", "Required Qty", "Check", "Notes"}

	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// 5. Table Body
	pdf.SetFont("Arial", "", 8)

	if len(items) == 0 {
		pdf.CellFormat(186, 10, "No procurement items added yet. Add items on screen or via AI Smart Paste.", "", 1, "C", false, 0, "")
	} else {
		for idx, item := range items {
			if idx%2 == 0 {
				pdf.SetFillColor(255, 255, 255)
			} else {
				pdf.SetFillColor(248, 250, 252)
			}

			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = "Procurement Item"
			}
			if len(name) > 50 {
				name = name[:47] + "..."
			}

			qty := item.Qty
			if qty == "" {
				qty = "1"
			}

			notes := item.Notes
			if notes == "" {
				notes = "Market Order"
			}

			pdf.SetTextColor(30, 41, 59)
			pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "", 0, "C", true, 0, "")
			pdf.CellFormat(colWidths[1], 6, fmt.Sprintf(" %s", name), "", 0, "L", true, 0, "")

			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(67, 56, 202)
			pdf.CellFormat(colWidths[2], 6, fmt.Sprintf(" %s", qty), "", 0, "C", true, 0, "")

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(100, 116, 139)
			pdf.CellFormat(colWidths[3], 6, "[   ]", "", 0, "C", true, 0, "")
			pdf.CellFormat(colWidths[4], 6, fmt.Sprintf(" %s", notes), "", 1, "L", true, 0, "")
		}
	}

	pdf.SetDrawColor(226, 232, 240)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())

	// 6. Footer Notes
	pdf.SetY(280)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 3, "Notes: Please verify stock availability and wholesale rates. Generated automatically by ShopMe Retail OS.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to output custom procurement PDF: %w", err)
	}

	return buf.Bytes(), nil
}
