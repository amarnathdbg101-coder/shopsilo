package utils

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// pdfSemaphore restricts max parallel PDF rendering goroutines to prevent CPU/memory spikes under heavy traffic.
var pdfSemaphore = make(chan struct{}, 15) // Max 15 concurrent PDF generation tasks

// RenderPDFWithConcurrencyLimit executes a single PDF generator function within a bounded concurrency semaphore pool.
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

type BatchPDFJob struct {
	ID         string
	RenderFunc func() ([]byte, error)
}

type BatchPDFResult struct {
	ID   string
	Data []byte
	Err  error
}

// RenderBatchPDFsConcurrently processes a slice of PDF render jobs concurrently using Goroutines & Channels.
// This is ideal for downloading multiple invoices or generating bulk reports in parallel.
func RenderBatchPDFsConcurrently(ctx context.Context, jobs []BatchPDFJob) ([]BatchPDFResult, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	results := make([]BatchPDFResult, len(jobs))
	resultsChan := make(chan struct {
		index int
		res   BatchPDFResult
	}, len(jobs))

	var wg sync.WaitGroup

	for i, job := range jobs {
		wg.Add(1)
		go func(idx int, j BatchPDFJob) {
			defer wg.Done()

			pdfBytes, err := RenderPDFWithConcurrencyLimit(ctx, j.RenderFunc)
			resultsChan <- struct {
				index int
				res   BatchPDFResult
			}{
				index: idx,
				res: BatchPDFResult{
					ID:   j.ID,
					Data: pdfBytes,
					Err:  err,
				},
			}
		}(i, job)
	}

	wg.Wait()
	close(resultsChan)

	for item := range resultsChan {
		results[item.index] = item.res
	}

	return results, nil
}
