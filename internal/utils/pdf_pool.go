package utils

import (
	"context"
	"fmt"
	"time"
)

// pdfSemaphore restricts max parallel PDF rendering goroutines to prevent CPU/memory spikes under heavy traffic.
var pdfSemaphore = make(chan struct{}, 10) // Max 10 concurrent PDF generation tasks

// RenderPDFWithConcurrencyLimit executes the given PDF generator function within a bounded concurrency semaphore pool.
func RenderPDFWithConcurrencyLimit(ctx context.Context, renderFunc func() ([]byte, error)) ([]byte, error) {
	select {
	case pdfSemaphore <- struct{}{}:
		defer func() { <-pdfSemaphore }()
		return renderFunc()
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("pdf generator service busy, please try again in a moment")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
