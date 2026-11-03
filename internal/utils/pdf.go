// Package utils provides helper utilities.
package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

// BuildWholesaleReorderHTML generates a clean A4 HTML template for low-stock restock orders with Hindi support.
func BuildWholesaleReorderHTML(shop *model.Shop, items []*dto.LowStockProduct) string {
	shopName := "ShopSilo Retail Store"
	storePhone := ""
	storeAddress := ""
	storeCity := ""
	storePincode := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			shopName = strings.TrimSpace(shop.Name)
		}
		storePhone = shop.Phone
		if shop.WhatsAppNumber != "" {
			storePhone = shop.WhatsAppNumber
		}
		storeAddress = shop.Address
		storeCity = shop.City
		storePincode = shop.Pincode
	}

	var rowsBuilder strings.Builder
	if len(items) == 0 {
		rowsBuilder.WriteString(`
			<tr>
				<td colspan="6" style="padding: 24px; text-align: center; color: #64748b; font-size: 12px;">
					Procurement list is empty. All store items are currently well-stocked.
				</td>
			</tr>
		`)
	} else {
		for i, item := range items {
			bg := "#ffffff"
			if i%2 == 1 {
				bg = "#f8fafc"
			}
			rowsBuilder.WriteString(fmt.Sprintf(`
				<tr style="background-color: %s;">
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #64748b;">%d</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; font-weight: 600; color: #0f172a;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #64748b;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; font-weight: 700; color: #dc2626;">%d</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; font-weight: 800; color: #4338ca;">%d</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; text-align: center; color: #94a3b8;">[ &nbsp; ]</td>
				</tr>
			`, bg, i+1, item.Name, item.SKU, item.CurrentStock, item.SuggestedReorderQty))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="hi">
<head>
	<meta charset="UTF-8">
	<title>Wholesale Re-Order Sheet</title>
	<style>
		@page { margin: 0; size: A4 portrait; }
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
	<div style="border-bottom: 2px solid #0f172a; padding-bottom: 12px; margin-bottom: 16px;">
		<div style="display: flex; justify-content: space-between; align-items: flex-start;">
			<div>
				<h1 style="margin: 0; font-size: 22px; font-weight: 800; color: #0f172a;">%s</h1>
				<div style="font-size: 11px; color: #64748b; margin-top: 3px;">%s, %s %s</div>
				%s
			</div>
			<div style="text-align: right;">
				<div style="font-size: 10px; font-weight: 800; color: #4338ca; text-transform: uppercase; background: #e0e7ff; padding: 4px 10px; border-radius: 4px; display: inline-block;">
					📦 MANDI & WHOLESALE PROCUREMENT
				</div>
				<div style="font-size: 11px; color: #64748b; margin-top: 6px;">Date: %s</div>
			</div>
		</div>
	</div>

	<!-- Summary Callout -->
	<div style="background: #eef2ff; border: 1.5px solid #c7d2fe; border-radius: 8px; padding: 10px 16px; margin-bottom: 16px; font-size: 12px; font-weight: 700; color: #3730a3; text-align: center;">
		TOTAL ITEMS REQUIRING MANDI / WHOLESALE RESTOCK: %d
	</div>

	<!-- Table -->
	<table style="width: 100%%; border-collapse: collapse; margin-bottom: 20px;">
		<thead>
			<tr style="background-color: #1e1b4b; color: #ffffff;">
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 30px;">#</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left;">Product Description (सामान का नाम)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 110px;">SKU / Code</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 100px;">Current Stock</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 110px;">Re-Order Qty</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 80px;">Check [ ]</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>

	<div style="border-top: 1px dashed #cbd5e1; padding-top: 12px; font-size: 10px; color: #94a3b8; text-align: center;">
		Generated automatically by ShopSilo Retail OS • Mandi Stock Procurement Sheet
	</div>
</body>
</html>`,
		shopName,
		storeAddress, storeCity, storePincode,
		formatPhoneHeader(storePhone),
		time.Now().Format("02-Jan-2006 15:04"),
		len(items),
		rowsBuilder.String(),
	)
}

// GenerateWholesaleReorderPDF generates an executive, ready-to-print PDF purchase order sheet for wholesale suppliers using chromedp.
func GenerateWholesaleReorderPDF(shop *model.Shop, items []*dto.LowStockProduct) ([]byte, error) {
	html := BuildWholesaleReorderHTML(shop, items)
	return GeneratePDFWithChromedp(context.Background(), html, DefaultA4Options())
}

// BuildCustomProcurementHTML generates an executive HTML template for custom screen procurement items with Hindi support.
func BuildCustomProcurementHTML(shop *model.Shop, title string, items []dto.ProcurementPDFItem) string {
	shopName := "ShopSilo Retail Store"
	storePhone := ""
	storeAddress := ""
	storeCity := ""
	storePincode := ""

	if shop != nil {
		if strings.TrimSpace(shop.Name) != "" {
			shopName = strings.TrimSpace(shop.Name)
		}
		storePhone = shop.Phone
		if shop.WhatsAppNumber != "" {
			storePhone = shop.WhatsAppNumber
		}
		storeAddress = shop.Address
		storeCity = shop.City
		storePincode = shop.Pincode
	}

	docTitle := "MANDI KHAREED & WHOLESALE PROCUREMENT SHEET"
	if strings.TrimSpace(title) != "" {
		docTitle = strings.ToUpper(strings.TrimSpace(title))
	}

	var rowsBuilder strings.Builder
	if len(items) == 0 {
		rowsBuilder.WriteString(`
			<tr>
				<td colspan="6" style="padding: 24px; text-align: center; color: #64748b; font-size: 12px;">
					No procurement items added yet.
				</td>
			</tr>
		`)
	} else {
		for i, it := range items {
			bg := "#ffffff"
			if i%2 == 1 {
				bg = "#f8fafc"
			}

			itemDesc := strings.TrimSpace(it.Name)
			if it.Category != "" {
				itemDesc = fmt.Sprintf("%s (%s)", it.Name, strings.TrimSpace(it.Category))
			}

			qty := it.Qty
			if qty == "" {
				qty = "1"
			}

			rate := strings.TrimSpace(it.EstimatedPrice)
			if rate == "" {
				rate = "-"
			}

			dealer := strings.TrimSpace(it.Dealer)
			if dealer == "" {
				dealer = "-"
			}

			rowsBuilder.WriteString(fmt.Sprintf(`
				<tr style="background-color: %s;">
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #64748b;">%d</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; font-weight: 600; color: #0f172a;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; font-weight: 700; color: #4338ca;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #334155;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; color: #475569;">%s</td>
					<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; text-align: center; color: #94a3b8;">[ &nbsp; ]</td>
				</tr>
			`, bg, i+1, itemDesc, qty, rate, dealer))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="hi">
<head>
	<meta charset="UTF-8">
	<title>%s</title>
	<style>
		@page { margin: 0; size: A4 portrait; }
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
	<div style="border-bottom: 2px solid #0f172a; padding-bottom: 12px; margin-bottom: 16px;">
		<div style="display: flex; justify-content: space-between; align-items: flex-start;">
			<div>
				<h1 style="margin: 0; font-size: 22px; font-weight: 800; color: #0f172a;">%s</h1>
				<div style="font-size: 11px; color: #64748b; margin-top: 3px;">%s, %s %s</div>
				%s
			</div>
			<div style="text-align: right;">
				<div style="font-size: 10px; font-weight: 800; color: #4338ca; text-transform: uppercase; background: #e0e7ff; padding: 4px 10px; border-radius: 4px; display: inline-block;">
					🛒 %s
				</div>
				<div style="font-size: 11px; color: #64748b; margin-top: 6px;">Date: %s</div>
			</div>
		</div>
	</div>

	<!-- Summary Callout -->
	<div style="background: #eef2ff; border: 1.5px solid #c7d2fe; border-radius: 8px; padding: 10px 16px; margin-bottom: 16px; font-size: 12px; font-weight: 700; color: #3730a3; text-align: center;">
		TOTAL ITEMS ON PROCUREMENT SHEET: %d
	</div>

	<!-- Table -->
	<table style="width: 100%%; border-collapse: collapse; margin-bottom: 20px;">
		<thead>
			<tr style="background-color: #1e1b4b; color: #ffffff;">
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 30px;">#</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left;">Item Description & Category</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 120px;">Required Qty</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 100px;">Est. Rate</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left; width: 140px;">Mandi Dealer / Wholesaler</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 80px;">Check [ ]</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>

	<div style="border-top: 1px dashed #cbd5e1; padding-top: 12px; font-size: 10px; color: #94a3b8; text-align: center;">
		Notes: Please verify stock availability and wholesale rates. Generated automatically by ShopSilo Retail OS.
	</div>
</body>
</html>`,
		docTitle,
		shopName,
		storeAddress, storeCity, storePincode,
		formatPhoneHeader(storePhone),
		docTitle,
		time.Now().Format("02-Jan-2006 15:04"),
		len(items),
		rowsBuilder.String(),
	)
}

// GenerateCustomProcurementPDF generates an executive PDF purchase order sheet directly from screen procurement items using chromedp.
func GenerateCustomProcurementPDF(shop *model.Shop, title string, items []dto.ProcurementPDFItem) ([]byte, error) {
	html := BuildCustomProcurementHTML(shop, title, items)
	return GeneratePDFWithChromedp(context.Background(), html, DefaultA4Options())
}
