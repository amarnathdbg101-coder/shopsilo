// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"

	"github.com/chromedp/chromedp"
	"github.com/signintech/gopdf"
	"go.uber.org/zap"
)

// maxConcurrentChromeRenders bounds simultaneous headless Chrome instances (each process consumes ~150-250MB RAM).
var chromeRenderSemaphore = make(chan struct{}, 2)

// SetChromeRenderSemaphoreConcurrency configures semaphore capacity (useful for tuning or tests).
func SetChromeRenderSemaphoreConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	chromeRenderSemaphore = make(chan struct{}, n)
}

// ChromedpPDFService renders pixel-perfect PDF reports using headless Chrome browser automation.
type ChromedpPDFService struct {
	shopRepo   *repository.ShopRepo
	posRepo    *repository.POSRepo
	drawerRepo *repository.CashDrawerRepo
	logger     *zap.Logger
}

// NewChromedpPDFService initializes ChromedpPDFService.
func NewChromedpPDFService(
	shopRepo *repository.ShopRepo,
	posRepo *repository.POSRepo,
	drawerRepo *repository.CashDrawerRepo,
	logger *zap.Logger,
) *ChromedpPDFService {
	return &ChromedpPDFService{
		shopRepo:   shopRepo,
		posRepo:    posRepo,
		drawerRepo: drawerRepo,
		logger:     logger,
	}
}

// isChromeInstalled checks if google-chrome or chromium is available on the host machine.
func isChromeInstalled() bool {
	candidates := []string{
		"google-chrome", "chrome", "chromium", "chromium-browser",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return true
		}
		if _, err := os.Stat(c); err == nil {
			return true
		}
	}
	return false
}

// GenerateEndOfDayPLPDF creates a high-definition executive Profit & Loss PDF report for the merchant.
func (s *ChromedpPDFService) GenerateEndOfDayPLPDF(ctx context.Context, shopOwnerUserID string, targetDate time.Time) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	dateStr := targetDate.Format("2006-01-02")
	formattedDate := targetDate.Format("02 Jan 2006")

	// Get POS day metrics
	summary, err := s.posRepo.GetDailyCloseReport(ctx, shop.ID, targetDate)
	if err != nil {
		summary = &dto.DailyCloseReport{
			Date: dateStr,
		}
	}

	totalSales := summary.TotalGrossSales
	expenses := summary.TotalExpensesPaid
	netProfit := totalSales - expenses
	if netProfit < 0 {
		netProfit = 0
	}

	cashShare := 0.0
	upiShare := 0.0
	creditShare := 0.0
	if totalSales > 0 {
		cashShare = (summary.CashSales / totalSales) * 100.0
		upiShare = (summary.UPISales / totalSales) * 100.0
		creditShare = (summary.CreditGiven / totalSales) * 100.0
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    margin: 0;
    padding: 32px;
    color: #0f172a;
    background-color: #ffffff;
  }
  .header {
    border-bottom: 2px solid #e2e8f0;
    padding-bottom: 20px;
    margin-bottom: 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .shop-title {
    font-size: 26px;
    font-weight: 800;
    color: #1e293b;
    margin: 0;
  }
  .report-tag {
    background: #e0e7ff;
    color: #4338ca;
    padding: 6px 14px;
    border-radius: 9999px;
    font-weight: 700;
    font-size: 13px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .grid-3 {
    display: flex;
    gap: 16px;
    margin-bottom: 24px;
  }
  .card {
    flex: 1;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    padding: 16px 20px;
  }
  .card.highlight {
    background: #ecfdf5;
    border-color: #a7f3d0;
  }
  .card.highlight .value {
    color: #059669;
  }
  .label {
    font-size: 13px;
    color: #64748b;
    font-weight: 600;
    margin-bottom: 6px;
    text-transform: uppercase;
  }
  .value {
    font-size: 24px;
    font-weight: 800;
    color: #0f172a;
  }
  .sub-metric {
    font-size: 12px;
    color: #64748b;
    margin-top: 4px;
  }
  table {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 16px;
    margin-bottom: 24px;
  }
  th {
    background-color: #f1f5f9;
    color: #475569;
    font-size: 13px;
    font-weight: 700;
    text-align: left;
    padding: 12px 16px;
    border-top: 1px solid #cbd5e1;
    border-bottom: 1px solid #cbd5e1;
  }
  td {
    padding: 12px 16px;
    border-bottom: 1px solid #f1f5f9;
    font-size: 14px;
    color: #334155;
  }
  .footer {
    border-top: 1px solid #e2e8f0;
    padding-top: 16px;
    margin-top: 32px;
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #94a3b8;
  }
</style>
</head>
<body>
  <div class="header">
    <div>
      <h1 class="shop-title">%s</h1>
      <p style="margin: 4px 0 0 0; color: #64748b; font-size: 14px;">Daily Financial & Profit Accountability Audit</p>
    </div>
    <div style="text-align: right;">
      <span class="report-tag">End-of-Day P&L</span>
      <p style="margin: 8px 0 0 0; font-weight: 700; color: #334155;">%s</p>
    </div>
  </div>

  <div class="grid-3">
    <div class="card">
      <div class="label">Gross Sales Total</div>
      <div class="value">₹%.2f</div>
      <div class="sub-metric">%d Bills Processed</div>
    </div>
    <div class="card">
      <div class="label">Expected Drawer Cash</div>
      <div class="value">₹%.2f</div>
      <div class="sub-metric">Reconciled in Drawer</div>
    </div>
    <div class="card highlight">
      <div class="label">Net Pocket Balance</div>
      <div class="value">₹%.2f</div>
      <div class="sub-metric">After Store Expenses</div>
    </div>
  </div>

  <h3 style="margin-bottom: 8px; font-size: 16px; color: #1e293b;">Counter Multi-Tender Breakdown</h3>
  <table>
    <thead>
      <tr>
        <th>Tender / Payment Mode</th>
        <th style="text-align: right;">Amount Collected</th>
        <th style="text-align: right;">Share</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td><strong>Physical Cash Tender</strong></td>
        <td style="text-align: right; font-weight: 700;">₹%.2f</td>
        <td style="text-align: right;">%.1f%%%%</td>
      </tr>
      <tr>
        <td><strong>Instant UPI / Online Pay</strong></td>
        <td style="text-align: right; font-weight: 700;">₹%.2f</td>
        <td style="text-align: right;">%.1f%%%%</td>
      </tr>
      <tr>
        <td><strong>Khata Credit Given (Udhar)</strong></td>
        <td style="text-align: right; font-weight: 700;">₹%.2f</td>
        <td style="text-align: right;">%.1f%%%%</td>
      </tr>
      <tr>
        <td><strong>Daily Store Operational Expenses</strong></td>
        <td style="text-align: right; font-weight: 700; color: #dc2626;">- ₹%.2f</td>
        <td style="text-align: right;">Deduction</td>
      </tr>
    </tbody>
  </table>

  <div class="footer">
    <span>ShopSilo Operating System • Automated Financial Engine</span>
    <span>Generated: %s</span>
  </div>
</body>
</html>`,
		shop.Name,
		formattedDate,
		totalSales, summary.TotalBillsCount,
		summary.ExpectedCashInDrawer,
		netProfit,
		summary.CashSales, cashShare,
		summary.UPISales, upiShare,
		summary.CreditGiven, creditShare,
		expenses,
		time.Now().Format("02-Jan-2006 15:04:05"),
	)

	// If Chrome is installed on the machine, render high-fidelity PDF via chromedp
	if isChromeInstalled() {
		pdfBytes, err := s.renderChromedpPDF(ctx, htmlContent)
		if err == nil && len(pdfBytes) > 0 {
			s.logger.Info("Chromedp rendered End-of-Day P&L PDF successfully", zap.Int("bytes", len(pdfBytes)))
			return pdfBytes, nil
		}
		s.logger.Warn("chromedp execution failed, falling back to gopdf", zap.Error(err))
	}

	// High-reliability fallback using Go native gopdf engine
	return s.renderFallbackGoPDF(shop.Name, formattedDate, summary, totalSales, netProfit)
}

// renderChromedpPDF executes Chrome headless print-to-pdf automation with concurrency bounding.
func (s *ChromedpPDFService) renderChromedpPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	// Guard: Bounded concurrency using semaphore channel to prevent VPS Out-of-Memory (OOM)
	select {
	case chromeRenderSemaphore <- struct{}{}:
		defer func() { <-chromeRenderSemaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for available Chrome rendering slot: %w", ctx.Err())
	}

	// Allocate Chrome execution context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	timeoutCtx, cancelTimeout := context.WithTimeout(chromeCtx, 25*time.Second)
	defer cancelTimeout()

	encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))
	dataURL := "data:text/html;base64," + encodedHTML

	err := chromedp.Do(timeoutCtx,
		chromedp.Navigate(dataURL),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return nil, err
	}

	pdfBuffer, err := chromedp.Run(timeoutCtx, chromedp.PrintToPDF(
		chromedp.PDFPaper(chromedp.PaperA4),
		chromedp.PDFMargin(0.4),
		chromedp.PDFPrintBackground(),
	))
	if err != nil {
		return nil, err
	}

	if len(pdfBuffer) < 4 || string(pdfBuffer[:4]) != "%PDF" {
		return nil, fmt.Errorf("invalid PDF output: missing %%PDF header (bytes: %d)", len(pdfBuffer))
	}

	return pdfBuffer, nil
}

// RenderDirectHTML renders any raw HTML string into PDF using chromedp and semaphore concurrency controls.
func (s *ChromedpPDFService) RenderDirectHTML(ctx context.Context, htmlContent string) ([]byte, error) {
	if !isChromeInstalled() {
		return nil, errors.New("chrome binary not found on host machine")
	}
	return s.renderChromedpPDF(ctx, htmlContent)
}

// renderFallbackGoPDF renders clean PDF using Go standard gopdf when Chrome binary is omitted.
func (s *ChromedpPDFService) renderFallbackGoPDF(
	shopName, dateStr string,
	summary *dto.DailyCloseReport,
	totalSales, netProfit float64,
) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	_ = pdf.AddTTFFont("roboto", "internal/utils/fonts/Roboto-Regular.ttf")
	_ = pdf.SetFont("roboto", "", 16)

	pdf.SetX(40)
	pdf.SetY(40)
	_ = pdf.Cell(nil, fmt.Sprintf("%s - Daily P&L Audit (%s)", shopName, dateStr))

	pdf.SetX(40)
	pdf.SetY(80)
	_ = pdf.Cell(nil, fmt.Sprintf("Gross Sales: Rs. %.2f (%d bills)", totalSales, summary.TotalBillsCount))

	pdf.SetX(40)
	pdf.SetY(110)
	_ = pdf.Cell(nil, fmt.Sprintf("Expected Cash in Drawer: Rs. %.2f", summary.ExpectedCashInDrawer))

	pdf.SetX(40)
	pdf.SetY(140)
	_ = pdf.Cell(nil, fmt.Sprintf("Net Pocket Balance: Rs. %.2f", netProfit))

	pdf.SetX(40)
	pdf.SetY(180)
	_ = pdf.Cell(nil, fmt.Sprintf("Cash Tender: Rs. %.2f | UPI: Rs. %.2f | Khata: Rs. %.2f", summary.CashSales, summary.UPISales, summary.CreditGiven))

	return pdf.GetBytesPdfReturnErr()
}
