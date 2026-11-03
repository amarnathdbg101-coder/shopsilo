// Package utils provides helper utilities.
package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"shopMe/internal/handler/model"
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

// BuildKhataStatementHTML generates an ultra-clean, executive A4 Khata passbook HTML template with full Hindi & Emoji support.
func BuildKhataStatementHTML(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) string {
	storeName := "ShopSilo Retail Store"
	storePhone := ""
	storeAddress := ""
	storeCity := ""
	storePincode := ""
	storeUPI := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			storeName = strings.TrimSpace(shop.Name)
		}
		storePhone = shop.Phone
		if shop.WhatsAppNumber != "" {
			storePhone = shop.WhatsAppNumber
		}
		storeAddress = shop.Address
		storeCity = shop.City
		storePincode = shop.Pincode
		storeUPI = shop.UPIID
	}

	custName := "Customer"
	custPhone := ""
	currentDue := 0.0
	if customer != nil {
		if strings.TrimSpace(customer.CustomerName) != "" {
			custName = strings.TrimSpace(customer.CustomerName)
		}
		custPhone = maskPhone(customer.CustomerMobile)
		currentDue = customer.CurrentBalance
	}

	// Calculate Totals
	totalDebit := 0.0  // Udhar
	totalCredit := 0.0 // Jama
	for _, tx := range transactions {
		if tx.Type == "GIVE_CREDIT" || tx.Type == "debit" || tx.Type == "DEBIT" {
			totalDebit += tx.Amount
		} else if tx.Type == "RECEIVE_PAYMENT" || tx.Type == "credit" || tx.Type == "CREDIT" {
			totalCredit += tx.Amount
		}
	}

	// Generate optional UPI QR code for outstanding due payment
	qrImgTag := ""
	if storeUPI != "" && currentDue > 0 {
		upiPayload := fmt.Sprintf("upi://pay?pa=%s&pn=%s&am=%.2f&cu=INR", storeUPI, storeName, currentDue)
		if qrBytes, err := GenerateQRCodePNG(upiPayload, 130); err == nil {
			qrBase64 := base64.StdEncoding.EncodeToString(qrBytes)
			qrImgTag = fmt.Sprintf(`
				<div style="display: flex; align-items: center; gap: 12px; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px 14px;">
					<img src="data:image/png;base64,%s" alt="UPI QR" style="width: 70px; height: 70px; border-radius: 4px;" />
					<div>
						<div style="font-size: 11px; font-weight: 700; color: #1e1b4b;">Scan & Pay via UPI (फोनपे / गूगलपे)</div>
						<div style="font-size: 10px; color: #64748b; margin-top: 2px;">UPI ID: %s</div>
						<div style="font-size: 9px; color: #16a34a; font-weight: 600; margin-top: 2px;">Instant Due Clearance</div>
					</div>
				</div>
			`, qrBase64, storeUPI)
		}
	}

	// Build Transaction Rows
	var rowsBuilder strings.Builder
	if len(transactions) == 0 {
		rowsBuilder.WriteString(`
			<tr>
				<td colspan="6" style="padding: 24px; text-align: center; color: #94a3b8; font-size: 12px;">
					No transaction entries found in this khata account statement.
				</td>
			</tr>
		`)
	} else {
		for i, tx := range transactions {
			bg := "#ffffff"
			if i%2 == 1 {
				bg = "#f8fafc"
			}

			dateStr := tx.CreatedAt.Format("02-01-2006 15:04")
			desc := strings.TrimSpace(tx.Notes)
			if desc == "" {
				desc = "General Khata Entry"
			}
			if tx.BillNumber != "" {
				desc = fmt.Sprintf("%s (Bill: #%s)", desc, tx.BillNumber)
			}

			debitStr := "-"
			creditStr := "-"
			if tx.Type == "GIVE_CREDIT" || tx.Type == "debit" || tx.Type == "DEBIT" {
				debitStr = fmt.Sprintf("&#8377;%.2f", tx.Amount)
			} else {
				creditStr = fmt.Sprintf("&#8377;%.2f", tx.Amount)
			}

			rowsBuilder.WriteString(fmt.Sprintf(`
				<tr style="background-color: %s;">
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #64748b;">%d</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; color: #475569;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; font-weight: 600; color: #0f172a;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; font-weight: 700; text-align: right; color: #dc2626;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; font-weight: 700; text-align: right; color: #16a34a;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; font-weight: 700; text-align: right; color: #1e1b4b;">&#8377;%.2f</td>
				</tr>
			`, bg, i+1, dateStr, desc, debitStr, creditStr, tx.BalanceAfter))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="hi">
<head>
	<meta charset="UTF-8">
	<title>Khata Statement - %s</title>
	<style>
		@page {
			margin: 0;
			size: A4 portrait;
		}
		* { box-sizing: border-box; }
		body {
			font-family: 'Nirmala UI', 'Segoe UI', 'Segoe UI Emoji', 'Noto Sans Devanagari', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
			margin: 0;
			padding: 28px;
			color: #0f172a;
			background-color: #ffffff;
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
	</style>
</head>
<body>
	<!-- Header -->
	<div style="border-bottom: 2px solid #0f172a; padding-bottom: 12px; margin-bottom: 16px;">
		<div style="display: flex; justify-content: space-between; align-items: flex-start;">
			<div>
				<h1 style="margin: 0; font-size: 22px; font-weight: 800; color: #0f172a; letter-spacing: -0.5px;">%s</h1>
				<div style="font-size: 11px; color: #64748b; margin-top: 3px;">
					%s, %s %s
				</div>
				%s
			</div>
			<div style="text-align: right;">
				<div style="font-size: 10px; font-weight: 800; color: #4338ca; text-transform: uppercase; letter-spacing: 0.5px; background: #e0e7ff; padding: 4px 10px; border-radius: 4px; display: inline-block;">
					📒 OFFICIAL KHATA PASSBOOK / खाता विवरण
				</div>
				<div style="font-size: 11px; color: #64748b; margin-top: 6px;">Generated: %s</div>
			</div>
		</div>
	</div>

	<!-- Customer Info & Summary Cards -->
	<div style="display: grid; grid-template-columns: 1.5fr 1fr 1fr 1.2fr; gap: 12px; margin-bottom: 20px;">
		<!-- Customer Profile Box -->
		<div style="background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 12px 14px;">
			<div style="font-size: 10px; color: #64748b; text-transform: uppercase; font-weight: 700;">Customer (खाताधारक)</div>
			<div style="font-size: 14px; font-weight: 800; color: #0f172a; margin-top: 2px;">%s</div>
			<div style="font-size: 11px; color: #64748b; margin-top: 2px;">📱 %s</div>
		</div>

		<!-- Total Udhar (Debit) Card -->
		<div style="background: #fef2f2; border: 1px solid #fecaca; border-radius: 8px; padding: 12px 14px;">
			<div style="font-size: 10px; color: #991b1b; text-transform: uppercase; font-weight: 700;">Total Udhar (कुल सामान उधार)</div>
			<div style="font-size: 16px; font-weight: 800; color: #dc2626; margin-top: 4px;">&#8377;%.2f</div>
		</div>

		<!-- Total Jama (Credit) Card -->
		<div style="background: #f0fdf4; border: 1px solid #bbf7d0; border-radius: 8px; padding: 12px 14px;">
			<div style="font-size: 10px; color: #166534; text-transform: uppercase; font-weight: 700;">Total Jama (कुल भुगतान जमा)</div>
			<div style="font-size: 16px; font-weight: 800; color: #16a34a; margin-top: 4px;">&#8377;%.2f</div>
		</div>

		<!-- Net Due Card -->
		<div style="background: #eef2ff; border: 1.5px solid #c7d2fe; border-radius: 8px; padding: 12px 14px;">
			<div style="font-size: 10px; color: #3730a3; text-transform: uppercase; font-weight: 800;">Net Balance Due (बाकी रकम)</div>
			<div style="font-size: 17px; font-weight: 900; color: #1e1b4b; margin-top: 4px;">&#8377;%.2f</div>
		</div>
	</div>

	<!-- Transactions Ledger Table -->
	<table style="width: 100%%; border-collapse: collapse; margin-bottom: 20px;">
		<thead>
			<tr style="background-color: #1e1b4b; color: #ffffff;">
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 30px;">#</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left; width: 130px;">Date & Time (दिनांक)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left;">Description (विवरण)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: right; width: 95px;">उधार (-) / Debit</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: right; width: 95px;">जमा (+) / Credit</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: right; width: 100px;">शेष / Balance</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>

	<!-- Footer with UPI QR & Notes -->
	<div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px;">
		%s
		<div style="text-align: right; font-size: 10px; color: #94a3b8;">
			<div>Computer-generated digital account passbook.</div>
			<div>Powered by ShopSilo Retail OS • Secure Khata Ledger</div>
		</div>
	</div>
</body>
</html>`,
		custName,
		storeName,
		storeAddress, storeCity, storePincode,
		formatPhoneHeader(storePhone),
		time.Now().Format("02-Jan-2006 15:04"),
		custName, custPhone,
		totalDebit,
		totalCredit,
		currentDue,
		rowsBuilder.String(),
		qrImgTag,
	)
}

// GenerateKhataStatementPDF produces an executive-grade, official customer account passbook PDF using chromedp.
func GenerateKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	html := BuildKhataStatementHTML(customer, transactions, shop)
	return GeneratePDFWithChromedp(context.Background(), html, DefaultA4Options())
}
