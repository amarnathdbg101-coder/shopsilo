package services

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestChromedp_DirectHTMLRendering(t *testing.T) {
	if !isChromeInstalled() {
		t.Skip("Skipping chromedp live test: Chrome binary not found on host machine")
	}

	service := NewChromedpPDFService(nil, nil, nil, zap.NewNop())

	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: sans-serif; padding: 24px; color: #1e293b; }
    h1 { color: #0284c7; }
    .badge { background: #e0f2fe; color: #0369a1; padding: 4px 8px; border-radius: 4px; font-weight: bold; }
  </style>
</head>
<body>
  <h1>ShopSilo Retail OS</h1>
  <p>Automated P&amp;L Financial Report generated via Headless Chrome.</p>
  <div class="badge">Net Profit: Rs. 14,500.00</div>
</body>
</html>`

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pdfBytes, err := service.RenderDirectHTML(ctx, html)
	if err != nil {
		t.Fatalf("RenderDirectHTML failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("expected non-empty PDF bytes")
	}

	// Validate PDF Magic Header (%PDF-)
	if !strings.HasPrefix(string(pdfBytes[:5]), "%PDF-") {
		t.Fatalf("expected %%PDF- header, got: %s", string(pdfBytes[:10]))
	}

	t.Logf("✅ Successfully rendered %d bytes PDF via Headless Chrome!", len(pdfBytes))
}

func TestChromedp_SemaphoreConcurrencyBounding(t *testing.T) {
	// Restrict semaphore to 1 slot for deterministic testing
	SetChromeRenderSemaphoreConcurrency(1)
	defer SetChromeRenderSemaphoreConcurrency(2) // reset to 2 after test

	// Acquire the 1 available slot manually
	chromeRenderSemaphore <- struct{}{}

	service := NewChromedpPDFService(nil, nil, nil, zap.NewNop())

	// Attempt to render in another goroutine with very short timeout (should be blocked by semaphore)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := service.renderChromedpPDF(ctx, "<html><body>Test</body></html>")
	if err == nil {
		t.Fatalf("expected render to fail due to semaphore timeout, got nil error")
	}

	if !strings.Contains(err.Error(), "timed out waiting for available Chrome rendering slot") {
		t.Fatalf("expected semaphore timeout error, got: %v", err)
	}

	// Release the manually acquired slot
	<-chromeRenderSemaphore
}

func TestChromedp_ConcurrentRenders(t *testing.T) {
	if !isChromeInstalled() {
		t.Skip("Skipping chromedp concurrent test: Chrome binary not found on host machine")
	}

	SetChromeRenderSemaphoreConcurrency(2)
	service := NewChromedpPDFService(nil, nil, nil, zap.NewNop())

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	html := `<!DOCTYPE html><html><body><h3>Concurrent Render Test</h3></body></html>`

	// Launch 3 concurrent renders across the 2-slot semaphore
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			pdf, err := service.RenderDirectHTML(ctx, html)
			if err != nil {
				errCh <- err
				return
			}
			if len(pdf) < 100 || !strings.HasPrefix(string(pdf[:4]), "%PDF") {
				t.Errorf("worker %d produced invalid PDF", id)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent worker error: %v", err)
	}
}
