// Package utils provides utility functions for ShopSilo backend.
package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"shopMe/internal/handler/model"
)

// ChromedpPDFOptions controls page layout, dimensions and print settings for browser-based PDF generation.
type ChromedpPDFOptions struct {
	PaperWidthInches   float64 // e.g. 5.83 for A5, 3.15 for 80mm POS Thermal, 8.27 for A4
	PaperHeightInches  float64 // e.g. 8.27 for A5, 11.69 for A4, 0 for continuous roll
	MarginTopInches    float64
	MarginBottomInches float64
	MarginLeftInches   float64
	MarginRightInches  float64
	Landscape          bool
	PrintBackground    bool
}

// DefaultA5Options returns print settings tailored for executive A5 counter bills & standees (148mm x 210mm).
func DefaultA5Options() ChromedpPDFOptions {
	return ChromedpPDFOptions{
		PaperWidthInches:   5.83,
		PaperHeightInches:  8.27,
		MarginTopInches:    0.2,
		MarginBottomInches: 0.2,
		MarginLeftInches:   0.25,
		MarginRightInches:  0.25,
		PrintBackground:    true,
	}
}

// DefaultA4Options returns print settings tailored for standard A4 reports, statements, and procurement sheets (210mm x 297mm).
func DefaultA4Options() ChromedpPDFOptions {
	return ChromedpPDFOptions{
		PaperWidthInches:   8.27,
		PaperHeightInches:  11.69,
		MarginTopInches:    0.25,
		MarginBottomInches: 0.25,
		MarginLeftInches:   0.3,
		MarginRightInches:  0.3,
		PrintBackground:    true,
	}
}

// DefaultThermalOptions returns print settings tailored for 80mm counter POS thermal receipt rolls.
func DefaultThermalOptions() ChromedpPDFOptions {
	return ChromedpPDFOptions{
		PaperWidthInches:   3.15, // 80mm
		PaperHeightInches:  7.0,  // Adjusts dynamically based on items
		MarginTopInches:    0.1,
		MarginBottomInches: 0.1,
		MarginLeftInches:   0.1,
		MarginRightInches:  0.1,
		PrintBackground:    true,
	}
}

var (
	resolvedChromePathOnce sync.Once
	resolvedChromePath     string

	sharedAllocOnce   sync.Once
	sharedAllocCtx    context.Context
	sharedAllocCancel context.CancelFunc
)

// FindChromiumPath locates the executable path of Chrome, Chromium, or Microsoft Edge on the host OS.
func FindChromiumPath() string {
	resolvedChromePathOnce.Do(func() {
		candidates := []string{
			"chrome",
			"chrome.exe",
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}

		for _, name := range candidates {
			if path, err := exec.LookPath(name); err == nil {
				resolvedChromePath = path
				return
			}
			if _, err := os.Stat(name); err == nil {
				resolvedChromePath = name
				return
			}
		}
	})
	return resolvedChromePath
}

// CheckChromiumAvailable verifies if a headless browser executable is discoverable on the host OS.
func CheckChromiumAvailable() bool {
	return FindChromiumPath() != ""
}

// getSharedBrowserAllocator returns a high-performance persistent browser allocator singleton.
// By keeping one warm browser master process alive, subsequent PDF requests avoid the ~1s cold start delay.
func getSharedBrowserAllocator(chromePath string) context.Context {
	sharedAllocOnce.Do(func() {
		allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.Flag("headless", "new"),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("disable-sync", true),
			chromedp.Flag("no-first-run", true),
			chromedp.Flag("hide-scrollbars", true),
			chromedp.Flag("mute-audio", true),
		)
		sharedAllocCtx, sharedAllocCancel = chromedp.NewExecAllocator(context.Background(), allocOpts...)
	})
	return sharedAllocCtx
}

// GeneratePDFWithChromedp renders raw HTML into a high-fidelity PDF using headless browser automation.
// It uses a warm persistent browser allocator, bounded timeout, and local file navigation for sub-second speeds.
func GeneratePDFWithChromedp(ctx context.Context, htmlContent string, opts ChromedpPDFOptions) ([]byte, error) {
	chromePath := FindChromiumPath()
	if chromePath == "" {
		return nil, fmt.Errorf("chromium binary not found on host system (please ensure Google Chrome or Microsoft Edge is installed)")
	}

	// 1. Write HTML to isolated temp file for instant, reliable local file navigation
	tmpFile, err := os.CreateTemp("", "shopsilo-pdf-*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp html file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write html to temp file: %w", err)
	}
	_ = tmpFile.Close()

	fileURL := "file:///" + filepath.ToSlash(tmpFile.Name())

	// 2. Obtain warm shared browser allocator for maximum performance (sub-second render)
	allocCtx := getSharedBrowserAllocator(chromePath)

	// 3. Create isolated tab context with a bounded 15-second timeout
	taskCtx, cancelTask := context.WithTimeout(allocCtx, 15*time.Second)
	defer cancelTask()

	browserCtx, cancelBrowser := chromedp.NewContext(taskCtx)
	defer cancelBrowser()

	// 4. Configure PDF print options
	pdfOptions := []chromedp.PDFOption{
		chromedp.PDFPrintBackground(),
		chromedp.PDFPaper(chromedp.PaperSize{
			Width:  opts.PaperWidthInches,
			Height: opts.PaperHeightInches,
		}),
		chromedp.PDFMargins(
			opts.MarginTopInches,
			opts.MarginRightInches,
			opts.MarginBottomInches,
			opts.MarginLeftInches,
		),
	}
	if opts.Landscape {
		pdfOptions = append(pdfOptions, chromedp.PDFLandscape())
	}

	// 5. Navigate to file URL (Navigate automatically waits for page load event)
	if err := chromedp.Do(browserCtx, chromedp.Navigate(fileURL)); err != nil {
		return nil, fmt.Errorf("chromedp navigation failed: %w", err)
	}

	// 6. Generate PDF bytes via Chrome DevTools Protocol
	pdfBuffer, err := chromedp.Run(browserCtx, chromedp.PrintToPDF(pdfOptions...))
	if err != nil {
		return nil, fmt.Errorf("chromedp print to pdf failed: %w", err)
	}

	return pdfBuffer, nil
}

// BuildInvoiceHTML generates an ultra-polished, bilingual HTML5 invoice template with full Hindi & Emoji support.
func BuildInvoiceHTML(bill *model.POSBill) string {
	storeName := "ShopSilo Retail Store"
	storePhone := ""
	storeAddress := ""
	storeCity := ""
	storePincode := ""

	if bill.Shop != nil {
		if strings.TrimSpace(bill.Shop.Name) != "" {
			storeName = strings.TrimSpace(bill.Shop.Name)
		}
		storePhone = bill.Shop.Phone
		if bill.Shop.WhatsAppNumber != "" {
			storePhone = bill.Shop.WhatsAppNumber
		}
		storeAddress = bill.Shop.Address
		storeCity = bill.Shop.City
		storePincode = bill.Shop.Pincode
	}

	customerDisplay := "Walk-in Customer (काउंटर ग्राहक)"
	if bill.CustomerPhone != "" {
		customerDisplay = maskPhone(bill.CustomerPhone)
	}

	// QR Code Generation for digital bill verification
	qrPayload := fmt.Sprintf("INVOICE:%s|AMT:%.2f|SHOP:%s", bill.BillNumber, bill.TotalAmount, storeName)
	qrBase64 := ""
	if qrBytes, err := GenerateQRCodePNG(qrPayload, 140); err == nil {
		qrBase64 = base64.StdEncoding.EncodeToString(qrBytes)
	}

	// Build Item Rows HTML
	var rowsBuilder strings.Builder
	for i, it := range bill.Items {
		bg := "#ffffff"
		if i%2 == 1 {
			bg = "#f8fafc"
		}
		rowsBuilder.WriteString(fmt.Sprintf(`
			<tr style="background-color: %s;">
				<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #64748b;">%d</td>
				<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; font-weight: 600; color: #0f172a;">
					%s
					%s
				</td>
				<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: center; color: #334155; font-weight: 600;">%d</td>
				<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 11px; text-align: right; color: #334155;">&#8377;%.2f</td>
				<td style="padding: 8px 10px; border-bottom: 1px solid #e2e8f0; font-size: 12px; font-weight: 700; text-align: right; color: #0f172a;">&#8377;%.2f</td>
			</tr>
		`, bg, i+1, it.ProductName, formatSKUBadge(it.ProductSKU), it.Quantity, it.UnitPrice, it.TotalPrice))
	}

	// Split Tender Breakdown
	var splitHTML strings.Builder
	if bill.PaymentMethod == "split" || bill.CashAmount > 0 || bill.OnlineAmount > 0 || bill.KhataAmount > 0 {
		splitHTML.WriteString(`<div style="margin-top: 10px; padding: 10px 14px; background-color: #f1f5f9; border-radius: 8px; font-size: 11px;">`)
		if bill.CashAmount > 0 {
			splitHTML.WriteString(fmt.Sprintf(`<div style="display:flex; justify-content:space-between; margin-bottom:4px;"><span style="color:#64748b;">💵 Paid in Cash (नकद):</span><span style="font-weight:700;">&#8377;%.2f</span></div>`, bill.CashAmount))
		}
		if bill.OnlineAmount > 0 {
			splitHTML.WriteString(fmt.Sprintf(`<div style="display:flex; justify-content:space-between; margin-bottom:4px;"><span style="color:#64748b;">💳 UPI / Online (ऑनलाइन):</span><span style="font-weight:700; color:#2563eb;">&#8377;%.2f</span></div>`, bill.OnlineAmount))
		}
		if bill.KhataAmount > 0 {
			splitHTML.WriteString(fmt.Sprintf(`<div style="display:flex; justify-content:space-between;"><span style="color:#dc2626; font-weight:600;">📒 Khata Due (उधार):</span><span style="font-weight:800; color:#dc2626;">&#8377;%.2f</span></div>`, bill.KhataAmount))
		}
		splitHTML.WriteString(`</div>`)
	}

	discountRow := ""
	if bill.DiscountAmount > 0 {
		discountRow = fmt.Sprintf(`
			<div style="display: flex; justify-content: space-between; font-size: 12px; color: #16a34a; font-weight: 600; margin-bottom: 4px;">
				<span>🏷️ Discount Savings (छूट बचत):</span>
				<span>- &#8377;%.2f</span>
			</div>
		`, bill.DiscountAmount)
	}

	qrImgTag := ""
	if qrBase64 != "" {
		qrImgTag = fmt.Sprintf(`<img src="data:image/png;base64,%s" alt="Invoice QR" style="width: 75px; height: 75px; border-radius: 6px; border: 1px solid #e2e8f0; padding: 2px;" />`, qrBase64)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="hi">
<head>
	<meta charset="UTF-8">
	<title>Invoice - %s</title>
	<style>
		@page {
			margin: 0;
			size: A5 portrait;
		}
		* { box-sizing: border-box; }
		body {
			font-family: 'Nirmala UI', 'Segoe UI', 'Noto Sans Devanagari', 'Segoe UI Emoji', 'Noto Color Emoji', 'Apple Color Emoji', -apple-system, BlinkMacSystemFont, Arial, sans-serif;
			margin: 0;
			padding: 24px;
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
				<h1 style="margin: 0; font-size: 20px; font-weight: 800; color: #0f172a; letter-spacing: -0.5px;">%s</h1>
				<div style="font-size: 11px; color: #64748b; margin-top: 3px;">
					%s, %s %s
				</div>
				%s
			</div>
			<div style="text-align: right;">
				<div style="font-size: 9px; font-weight: 800; color: #4338ca; text-transform: uppercase; letter-spacing: 0.5px; background: #e0e7ff; padding: 3px 8px; border-radius: 4px; display: inline-block;">
					🧾 TAX INVOICE / बिल
				</div>
				<div style="font-size: 13px; font-weight: 700; color: #0f172a; margin-top: 4px;">#%s</div>
				<div style="font-size: 11px; color: #64748b;">%s</div>
			</div>
		</div>
	</div>

	<!-- Metadata Bar -->
	<div style="display: flex; justify-content: space-between; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 8px 14px; margin-bottom: 16px; font-size: 11px;">
		<div><span style="color: #64748b;">Customer (ग्राहक):</span> <strong style="color: #0f172a;">%s</strong></div>
		<div><span style="color: #64748b;">Payment (भुगतान):</span> <strong style="color: #4338ca; text-transform: uppercase;">%s</strong></div>
	</div>

	<!-- Items Table -->
	<table style="width: 100%%; border-collapse: collapse; margin-bottom: 16px;">
		<thead>
			<tr style="background-color: #1e1b4b; color: #ffffff;">
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 30px;">#</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: left;">Item Description (सामान का विवरण)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: center; width: 45px;">Qty (मात्रा)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: right; width: 75px;">Rate (दर)</th>
				<th style="padding: 8px 10px; font-size: 10px; font-weight: 700; text-align: right; width: 90px;">Amount (रकम)</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>

	<!-- Summary & Totals -->
	<div style="display: flex; justify-content: space-between; align-items: flex-end; margin-top: 12px;">
		<div style="display: flex; align-items: center; gap: 12px;">
			%s
			<div>
				<div style="font-size: 10px; font-weight: 600; color: #4338ca;">Scan QR to verify bill</div>
				<div style="font-size: 9px; color: #94a3b8; margin-top: 2px;">ShopSilo Digital Retail OS</div>
			</div>
		</div>

		<div style="width: 230px;">
			<div style="display: flex; justify-content: space-between; font-size: 12px; color: #64748b; margin-bottom: 4px;">
				<span>Subtotal (योग):</span>
				<span style="font-weight: 600; color: #334155;">&#8377;%.2f</span>
			</div>
			%s
			<div style="display: flex; justify-content: space-between; font-size: 15px; font-weight: 800; color: #1e1b4b; background: #e0e7ff; padding: 7px 12px; border-radius: 6px; margin-top: 6px;">
				<span>Grand Total (कुल देय):</span>
				<span>&#8377;%.2f</span>
			</div>
			%s
		</div>
	</div>

	<!-- Footer Note -->
	<div style="margin-top: 24px; text-align: center; border-top: 1px dashed #cbd5e1; padding-top: 12px;">
		<div style="font-size: 11px; font-weight: 700; color: #4338ca;">धन्यवाद! फिर पधारें! • Thank you for shopping with us!</div>
		<div style="font-size: 9px; color: #94a3b8; margin-top: 3px;">This is a computer-generated digital tax invoice. Powered by ShopSilo.</div>
	</div>
</body>
</html>`,
		bill.BillNumber,
		storeName,
		storeAddress, storeCity, storePincode,
		formatPhoneHeader(storePhone),
		bill.BillNumber,
		bill.CreatedAt.Format("02 Jan 2006, 03:04 PM"),
		customerDisplay,
		bill.PaymentMethod,
		rowsBuilder.String(),
		qrImgTag,
		bill.Subtotal,
		discountRow,
		bill.TotalAmount,
		splitHTML.String(),
	)
}

func formatPhoneHeader(phone string) string {
	if phone == "" {
		return ""
	}
	return fmt.Sprintf(`<div style="font-size: 11px; color: #64748b;">Phone / WhatsApp: %s</div>`, phone)
}

func formatSKUBadge(sku string) string {
	if sku == "" {
		return ""
	}
	return fmt.Sprintf(`<div style="font-size: 9px; color: #94a3b8; font-weight: 500;">SKU: %s</div>`, sku)
}

// GenerateSmartBillPDF generates an invoice PDF using headless browser automation with full Hindi and Unicode support.
func GenerateSmartBillPDF(ctx context.Context, bill *model.POSBill) ([]byte, error) {
	html := BuildInvoiceHTML(bill)
	return GeneratePDFWithChromedp(ctx, html, DefaultA5Options())
}
