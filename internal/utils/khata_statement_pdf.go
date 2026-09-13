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

// GenerateKhataStatementPDF produces an executive-grade, official customer account passbook PDF.
func GenerateKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "") // Standard A4 ledger sheet
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	// 1. Header (Store Branding)
	pdf.SetFillColor(15, 23, 42) // Slate Navy Accent Header
	pdf.Rect(0, 0, 210, 24, "F")

	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(255, 255, 255)
	storeName := "SHOPME RETAIL STORE"
	if shop != nil && shop.Name != "" {
		storeName = strings.ToUpper(shop.Name)
	}
	pdf.SetY(6)
	pdf.CellFormat(0, 7, storeName, "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(199, 210, 254)
	pdf.CellFormat(0, 4, "OFFICIAL KHATA PASSBOOK & ACCOUNT STATEMENT", "", 1, "C", false, 0, "")

	pdf.SetY(28)

	// 2. Store Address & Contact Info
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
	pdf.Ln(2)

	// Divider line
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())
	pdf.Ln(4)

	// 3. Customer Info Section
	metaY := pdf.GetY()
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.RoundedRect(12, metaY, 186, 16, 2, "1234", "FD")

	pdf.SetY(metaY + 2)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(93, 4, fmt.Sprintf(" Customer Name: %s", customer.CustomerName), "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 4, fmt.Sprintf("Mobile: %s ", maskPhone(customer.CustomerMobile)), "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(93, 4, fmt.Sprintf(" Statement Generated: %s", customer.UpdatedAt.Format("02-Jan-2006 15:04")), "", 0, "L", false, 0, "")

	limitStr := "No Cap"
	if customer.CreditLimit > 0 {
		limitStr = fmt.Sprintf("Rs. %.2f", customer.CreditLimit)
	}
	pdf.CellFormat(90, 4, fmt.Sprintf("Approved Credit Limit: %s ", limitStr), "", 1, "R", false, 0, "")

	pdf.Ln(6)

	// 4. Summary Status Callout Badge
	sumY := pdf.GetY()
	if customer.CurrentBalance > 0 {
		pdf.SetFillColor(254, 242, 242) // Light Crimson Red
		pdf.SetDrawColor(252, 165, 165)
		pdf.SetTextColor(220, 38, 38)
	} else {
		pdf.SetFillColor(240, 253, 244) // Light Emerald Green
		pdf.SetDrawColor(134, 239, 172)
		pdf.SetTextColor(22, 163, 74)
	}
	pdf.RoundedRect(12, sumY, 186, 12, 2, "1234", "FD")

	pdf.SetY(sumY + 2)
	pdf.SetFont("Arial", "B", 10)

	balanceText := fmt.Sprintf("TOTAL OUTSTANDING DUE: Rs. %.2f", customer.CurrentBalance)
	if customer.CurrentBalance <= 0 {
		balanceText = "TOTAL OUTSTANDING DUE: NIL (ALL CLEAR)"
	}
	pdf.CellFormat(186, 8, balanceText, "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// 5. Ledger Table Header
	pdf.SetFillColor(67, 56, 202) // Deep Indigo Header
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(255, 255, 255)

	colWidths := []float64{10, 32, 28, 50, 22, 22, 22}
	pdf.CellFormat(colWidths[0], 7, "#", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[1], 7, "Date & Time", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[2], 7, "Tx Type", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[3], 7, "Description / Notes", "", 0, "L", true, 0, "")
	pdf.CellFormat(colWidths[4], 7, "Mode", "", 0, "C", true, 0, "")
	pdf.CellFormat(colWidths[5], 7, "Amount", "", 0, "R", true, 0, "")
	pdf.CellFormat(colWidths[6], 7, "Balance ", "", 1, "R", true, 0, "")

	// 6. Zebra Striped Ledger Transactions
	pdf.SetFont("Arial", "", 8)

	for idx, tx := range transactions {
		if idx%2 == 0 {
			pdf.SetFillColor(255, 255, 255)
		} else {
			pdf.SetFillColor(248, 250, 252) // Subtle Zebra Stripe
		}

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
		if len(notes) > 32 {
			notes = notes[:29] + "..."
		}
		if notes == "" {
			notes = "-"
		}

		mode := strings.ToUpper(tx.PaymentMode)
		if mode == "" {
			mode = "-"
		}

		pdf.SetTextColor(30, 41, 59)
		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", idx+1), "", 0, "C", true, 0, "")
		pdf.CellFormat(colWidths[1], 6, tx.CreatedAt.Format("02-Jan-06 15:04"), "", 0, "C", true, 0, "")

		// Highlight Tx Type
		if tx.Type == model.KhataTxTypeReceivePayment {
			pdf.SetTextColor(22, 163, 74) // Green for Jama
		} else {
			pdf.SetTextColor(220, 38, 38) // Red for Udhar
		}
		pdf.CellFormat(colWidths[2], 6, txType, "", 0, "C", true, 0, "")

		pdf.SetTextColor(30, 41, 59)
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf(" %s", notes), "", 0, "L", true, 0, "")
		pdf.CellFormat(colWidths[4], 6, mode, "", 0, "C", true, 0, "")
		pdf.CellFormat(colWidths[5], 6, fmt.Sprintf("%.2f ", tx.Amount), "", 0, "R", true, 0, "")

		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(colWidths[6], 6, fmt.Sprintf("%.2f ", tx.BalanceAfter), "", 1, "R", true, 0, "")
		pdf.SetFont("Arial", "", 8)
	}

	pdf.SetDrawColor(226, 232, 240)
	pdf.Line(12, pdf.GetY(), 198, pdf.GetY())

	// 7. Dynamic UPI Payment Collect QR Code
	if customer.CurrentBalance > 0 && shop != nil {
		qrPayload := fmt.Sprintf("upi://pay?pa=%s&pn=%s&am=%.2f&cu=INR", shop.Phone, storeName, customer.CurrentBalance)
		qrBytes, err := GenerateQRCodePNG(qrPayload, 120)
		if err == nil && len(qrBytes) > 0 {
			qrHolder := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
			pdf.RegisterImageOptionsReader("khata_qr", qrHolder, bytes.NewReader(qrBytes))

			qrY := pdf.GetY() + 4
			if qrY < 240 {
				pdf.SetY(qrY)
				pdf.ImageOptions("khata_qr", 12, qrY, 22, 22, false, qrHolder, 0, "")
				pdf.SetX(38)
				pdf.SetFont("Arial", "B", 8)
				pdf.SetTextColor(67, 56, 202)
				pdf.CellFormat(0, 4, "SCAN TO PAY & CLEAR KHATA DUE VIA UPI", "", 1, "L", false, 0, "")
				pdf.SetX(38)
				pdf.SetFont("Arial", "", 7)
				pdf.SetTextColor(100, 116, 139)
				pdf.CellFormat(0, 4, "Scan this QR code with PhonePe, Google Pay, or Paytm to pay directly to store counter.", "", 1, "L", false, 0, "")
			}
		}
	}

	// 8. Footer
	pdf.SetY(280)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(0, 3, "Computer-generated official account statement. Verified by ShopMe Retail OS.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate statement PDF: %w", err)
	}

	return buf.Bytes(), nil
}
