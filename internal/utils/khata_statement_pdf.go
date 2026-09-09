// Package utils provides helper utilities.
package utils

import (
	"bytes"
	"fmt"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

func maskPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 4 {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "+") && len(trimmed) >= 12 {
		return trimmed[:5] + "*****" + trimmed[len(trimmed)-3:]
	}
	if len(trimmed) >= 10 {
		return trimmed[:2] + "*****" + trimmed[len(trimmed)-3:]
	}
	return trimmed[:1] + "***" + trimmed[len(trimmed)-2:]
}

// GenerateKhataStatementPDF produces an official, itemized customer account passbook PDF.
func GenerateKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "") // Standard A4 ledger sheet
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	// 1. Header (Store Branding)
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(30, 41, 59)
	storeName := "ShopMe Retail Store"
	if shop != nil && shop.Name != "" {
		storeName = shop.Name
	}
	pdf.CellFormat(0, 8, storeName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 116, 139)
	if shop != nil {
		if shop.Address != "" {
			pdf.CellFormat(0, 5, shop.Address, "", 1, "C", false, 0, "")
		}
		contact := shop.Phone
		if shop.WhatsAppNumber != "" {
			contact = fmt.Sprintf("Phone / WhatsApp: %s", shop.WhatsAppNumber)
		}
		if contact != "" {
			pdf.CellFormat(0, 5, contact, "", 1, "C", false, 0, "")
		}
	}
	pdf.Ln(3)

	// Divider line
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())
	pdf.Ln(4)

	// 2. Title & Customer Info
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(0, 6, "CUSTOMER KHATA PASSBOOK STATEMENT", "", 1, "C", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(51, 65, 85)
	pdf.CellFormat(93, 5, fmt.Sprintf("Customer Name: %s", customer.CustomerName), "", 0, "L", false, 0, "")
	pdf.CellFormat(93, 5, fmt.Sprintf("Mobile: %s", maskPhone(customer.CustomerMobile)), "", 1, "R", false, 0, "")

	pdf.CellFormat(93, 5, fmt.Sprintf("Statement Date: %s", customer.UpdatedAt.Format("02-Jan-2006 15:04")), "", 0, "L", false, 0, "")
	if customer.CreditLimit > 0 {
		pdf.CellFormat(93, 5, fmt.Sprintf("Credit Limit: Rs. %.2f", customer.CreditLimit), "", 1, "R", false, 0, "")
	} else {
		pdf.CellFormat(93, 5, "Credit Limit: No Cap", "", 1, "R", false, 0, "")
	}
	pdf.Ln(4)

	// 3. Summary Callout Box
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(203, 213, 225)
	pdf.RoundedRect(12, pdf.GetY(), 186, 14, 2, "1234", "FD")

	pdf.SetY(pdf.GetY() + 3)
	pdf.SetFont("Arial", "B", 11)
	if customer.CurrentBalance > 0 {
		pdf.SetTextColor(220, 38, 38) // Red for due
	} else {
		pdf.SetTextColor(22, 163, 74) // Green for all clear
	}
	balanceText := fmt.Sprintf("Total Outstanding Due: Rs. %.2f", customer.CurrentBalance)
	if customer.CurrentBalance <= 0 {
		balanceText = "Total Outstanding Due: Nil (All Clear)"
	}
	pdf.CellFormat(186, 8, balanceText, "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// 4. Ledger Table Header
	pdf.SetFillColor(241, 245, 249)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(30, 41, 59)

	colWidths := []float64{12, 32, 28, 48, 22, 22, 22}
	pdf.CellFormat(colWidths[0], 7, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[1], 7, "Date & Time", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[2], 7, "Tx Type", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[3], 7, "Description / Notes", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colWidths[4], 7, "Mode", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[5], 7, "Amount", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[6], 7, "Balance", "1", 1, "R", true, 0, "")

	// 5. Ledger Transactions Rows
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(51, 65, 85)

	for idx, tx := range transactions {
		txType := "Credit (Udhar)"
		if tx.Type == model.KhataTxTypeReceivePayment {
			txType = "Payment (Jama)"
		}

		notes := tx.Notes
		if tx.BillNumber != "" {
			if notes != "" {
				notes = fmt.Sprintf("Bill #%s - %s", tx.BillNumber, notes)
			} else {
				notes = fmt.Sprintf("Bill #%s", tx.BillNumber)
			}
		}
		if len(notes) > 30 {
			notes = notes[:27] + "..."
		}
		if notes == "" {
			notes = "-"
		}

		mode := strings.ToUpper(tx.PaymentMode)
		if mode == "" {
			mode = "-"
		}

		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, tx.CreatedAt.Format("02-Jan-06 15:04"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, txType, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf(" %s", notes), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, mode, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, fmt.Sprintf("%.2f ", tx.Amount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[6], 6, fmt.Sprintf("%.2f ", tx.BalanceAfter), "1", 1, "R", false, 0, "")
	}

	// 6. Footer
	pdf.Ln(8)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 4, "This is a computer generated account statement. Verified by ShopMe Retail OS.", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, "For any inquiries or discrepancies, please contact the store counter directly.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate statement PDF: %w", err)
	}

	return buf.Bytes(), nil
}
