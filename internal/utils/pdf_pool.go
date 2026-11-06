package utils

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// pdfSemaphore restricts max parallel PDF rendering goroutines to 5 to prevent CPU and memory spikes.
var pdfSemaphore = make(chan struct{}, 5) // Max 5 concurrent PDF generation tasks

type cachedPDFItem struct {
	data      []byte
	expiresAt time.Time
}

var (
	pdfCacheMu sync.RWMutex
	pdfCache   = make(map[string]cachedPDFItem)
)

// GetCachedPDF retrieves cached PDF bytes by key if still valid (sub-millisecond instant delivery).
func GetCachedPDF(key string) ([]byte, bool) {
	pdfCacheMu.RLock()
	defer pdfCacheMu.RUnlock()
	item, ok := pdfCache[key]
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.data, true
}

// SetCachedPDF stores rendered PDF bytes with a given TTL to avoid re-rendering under traffic surges.
func SetCachedPDF(key string, data []byte, ttl time.Duration) {
	if len(data) == 0 {
		return
	}
	pdfCacheMu.Lock()
	defer pdfCacheMu.Unlock()

	// Sweep expired entries when cache exceeds 500 items to avoid unbounded memory growth
	if len(pdfCache) > 500 {
		now := time.Now()
		for k, v := range pdfCache {
			if now.After(v.expiresAt) {
				delete(pdfCache, k)
			}
		}
	}

	pdfCache[key] = cachedPDFItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

// RenderPDFWithConcurrencyLimit executes a single PDF generator function within a bounded concurrency semaphore pool of 5 workers.
func RenderPDFWithConcurrencyLimit(ctx context.Context, renderFunc func() ([]byte, error)) ([]byte, error) {
	select {
	case pdfSemaphore <- struct{}{}:
		defer func() { <-pdfSemaphore }()
		return renderFunc()
	case <-time.After(15 * time.Second):
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
