// Package utils provides helper utilities.
package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"shopMe/internal/handler/model"
)

// BuildStoreStandeeHTML generates an ultra-polished, executive A5 counter standee HTML template with full Hindi & Emoji support.
func BuildStoreStandeeHTML(shop *model.Shop) string {
	storeName := "ShopSilo Retail Store"
	category := "General Retail Store"
	storeAddress := "Local Market"
	storeCity := ""
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
		storeCity = strings.TrimSpace(shop.City)
		if strings.TrimSpace(shop.WhatsAppNumber) != "" {
			storePhone = strings.TrimSpace(shop.WhatsAppNumber)
		} else if strings.TrimSpace(shop.Phone) != "" {
			storePhone = strings.TrimSpace(shop.Phone)
		}
		storeUPI = strings.TrimSpace(shop.UPIID)
	}

	qrPayload := fmt.Sprintf("https://shopsilo.in/store/%s", strings.ToLower(strings.ReplaceAll(storeName, " ", "-")))
	if shop != nil && shop.ID != "" {
		qrPayload = fmt.Sprintf("https://shopsilo.in/store/%s", shop.ID)
	}

	qrBase64 := ""
	if qrBytes, err := GenerateQRCodePNG(qrPayload, 260); err == nil {
		qrBase64 = base64.StdEncoding.EncodeToString(qrBytes)
	}

	qrImgTag := ""
	if qrBase64 != "" {
		qrImgTag = fmt.Sprintf(`<img src="data:image/png;base64,%s" alt="Store QR" style="width: 200px; height: 200px; display: block; margin: 0 auto; border-radius: 8px;" />`, qrBase64)
	}

	upiNote := ""
	if storeUPI != "" {
		upiNote = fmt.Sprintf(`<div style="font-size: 11px; font-weight: 700; color: #4338ca; margin-top: 4px;">💳 Accepted Payments: UPI (%s) / Cash</div>`, storeUPI)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="hi">
<head>
	<meta charset="UTF-8">
	<title>Store Standee - %s</title>
	<style>
		@page { margin: 0; size: A5 portrait; }
		* { box-sizing: border-box; }
		body {
			font-family: 'Nirmala UI', 'Segoe UI', 'Segoe UI Emoji', 'Noto Sans Devanagari', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
			margin: 0;
			padding: 24px;
			color: #0f172a;
			background-color: #ffffff;
			text-align: center;
			-webkit-print-color-adjust: exact;
			print-color-adjust: exact;
		}
	</style>
</head>
<body>
	<!-- Brand Pill -->
	<div style="background: #1e1b4b; color: #ffffff; padding: 6px 14px; border-radius: 20px; display: inline-block; font-size: 10px; font-weight: 800; letter-spacing: 0.5px; text-transform: uppercase; margin-bottom: 12px;">
		SHOPSILO LOCAL STOREFRONT • VERIFIED MERCHANT
	</div>

	<!-- Store Title -->
	<h1 style="font-size: 26px; font-weight: 900; color: #0f172a; margin: 0; letter-spacing: -0.5px;">%s</h1>
	<div style="display: inline-block; background: #e0e7ff; color: #4338ca; font-size: 11px; font-weight: 700; padding: 3px 10px; border-radius: 6px; margin-top: 6px;">
		%s
	</div>
	<div style="font-size: 11px; color: #64748b; margin-top: 6px;">
		📍 %s%s
	</div>

	<!-- Center QR Frame -->
	<div style="margin: 20px auto; padding: 16px; background: #f8fafc; border: 2px solid #e2e8f0; border-radius: 16px; width: 236px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
		%s
		<div style="margin-top: 10px; font-size: 12px; font-weight: 800; color: #1e1b4b; text-transform: uppercase; letter-spacing: 0.5px;">
			SCAN TO ORDER ONLINE
		</div>
		<div style="font-size: 11px; color: #4338ca; font-weight: 600; margin-top: 2px;">
			(QR स्कैन करके ऑनलाइन सामान मंगाएं)
		</div>
	</div>

	<!-- 3-Step Walkthrough Pills -->
	<div style="display: flex; justify-content: center; gap: 8px; margin-bottom: 20px;">
		<div style="background: #f1f5f9; padding: 6px 10px; border-radius: 6px; font-size: 10px; font-weight: 600; color: #334155;">
			📸 1. Scan with Phone
		</div>
		<div style="background: #f1f5f9; padding: 6px 10px; border-radius: 6px; font-size: 10px; font-weight: 600; color: #334155;">
			🛒 2. View Live Catalog
		</div>
		<div style="background: #f1f5f9; padding: 6px 10px; border-radius: 6px; font-size: 10px; font-weight: 600; color: #334155;">
			⚡ 3. Fast Pickup / Delivery
		</div>
	</div>

	<!-- Bottom Merchant Details -->
	<div style="border-top: 1.5px dashed #cbd5e1; padding-top: 12px; margin-top: 16px;">
		<div style="font-size: 12px; font-weight: 800; color: #0f172a;">
			📞 Order on WhatsApp: %s
		</div>
		%s
		<div style="font-size: 9px; color: #94a3b8; margin-top: 6px;">
			Powered by ShopSilo Retail OS • Official Digital Counter Standee
		</div>
	</div>
</body>
</html>`,
		storeName,
		storeName,
		category,
		storeAddress,
		formatCity(storeCity),
		qrImgTag,
		storePhone,
		upiNote,
	)
}

func formatCity(city string) string {
	if city == "" {
		return ""
	}
	return ", " + city
}

// GenerateStoreStandeePDF generates a professional, high-resolution printable A5 counter standee PDF for merchants using chromedp.
func GenerateStoreStandeePDF(shop *model.Shop) ([]byte, error) {
	html := BuildStoreStandeeHTML(shop)
	return GeneratePDFWithChromedp(context.Background(), html, DefaultA5Options())
}
