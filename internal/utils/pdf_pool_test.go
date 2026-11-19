package utils

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPDFCache_SetGetInvalidate(t *testing.T) {
	key := "test_pdf_1"
	sampleData := []byte("%PDF-1.4 sample pdf content")

	// 1. Initial Get should be false
	if _, found := GetCachedPDF(key); found {
		t.Fatalf("expected cache miss for non-existent key")
	}

	// 2. Set with short TTL
	SetCachedPDF(key, sampleData, 200*time.Millisecond)

	// 3. Immediate Get should succeed and match
	data, found := GetCachedPDF(key)
	if !found || string(data) != string(sampleData) {
		t.Fatalf("expected cache hit with matching data, got found=%v", found)
	}

	// 4. Invalidate specific key
	InvalidateCachedPDF(key)
	if _, found := GetCachedPDF(key); found {
		t.Fatalf("expected cache miss after InvalidateCachedPDF")
	}

	// 5. Prefix Invalidation test
	keyPrefix1 := "khata_stmt:shop123:cust456"
	keyPrefix2 := "khata_stmt:shop123:cust789"
	keyOther := "khata_stmt:shop999:cust111"

	SetCachedPDF(keyPrefix1, sampleData, 1*time.Minute)
	SetCachedPDF(keyPrefix2, sampleData, 1*time.Minute)
	SetCachedPDF(keyOther, sampleData, 1*time.Minute)

	InvalidateCachedPDFByPrefix("khata_stmt:shop123:")

	if _, found := GetCachedPDF(keyPrefix1); found {
		t.Errorf("expected keyPrefix1 to be invalidated")
	}
	if _, found := GetCachedPDF(keyPrefix2); found {
		t.Errorf("expected keyPrefix2 to be invalidated")
	}
	if _, found := GetCachedPDF(keyOther); !found {
		t.Errorf("expected keyOther to still exist in cache")
	}
}

func TestRenderPDFWithConcurrencyLimit(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup
	var activeRoutines int
	var maxObserved int
	var mu sync.Mutex

	numWorkers := 15
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			_, err := RenderPDFWithConcurrencyLimit(ctx, func() ([]byte, error) {
				mu.Lock()
				activeRoutines++
				if activeRoutines > maxObserved {
					maxObserved = activeRoutines
				}
				mu.Unlock()

				time.Sleep(30 * time.Millisecond)

				mu.Lock()
				activeRoutines--
				mu.Unlock()
				return []byte("%PDF-1.4 simulated"), nil
			})
			if err != nil {
				t.Errorf("RenderPDFWithConcurrencyLimit error: %v", err)
			}
		}()
	}

	wg.Wait()

	if maxObserved > cap(pdfSemaphore) {
		t.Fatalf("max concurrent PDF workers exceeded capacity: observed %d, cap %d", maxObserved, cap(pdfSemaphore))
	}
}
