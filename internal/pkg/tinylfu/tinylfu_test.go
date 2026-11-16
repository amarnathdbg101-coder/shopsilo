package tinylfu_test

import (
	"fmt"
	"testing"

	"shopMe/internal/pkg/tinylfu"
)

func TestBasicGetSet(t *testing.T) {
	cache, err := tinylfu.New[string, int](10)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	cache.Set("apple", 100)
	val, ok := cache.Get("apple")
	if !ok || val != 100 {
		t.Fatalf("expected 100, got %d (ok=%v)", val, ok)
	}

	_, ok = cache.Get("banana")
	if ok {
		t.Fatalf("unexpected hit for non-existent key")
	}
}

func TestScanResistance(t *testing.T) {
	// Small cache of capacity 10
	cache, err := tinylfu.New[string, int](10)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	// 1. Establish high frequency for hot keys "hot1" and "hot2"
	for i := 0; i < 50; i++ {
		cache.Set("hot1", 1)
		cache.Set("hot2", 2)
		cache.Get("hot1")
		cache.Get("hot2")
	}

	// 2. Perform a massive sequential scan of 100 one-off cold keys
	// In naive LRU, this would completely evict "hot1" and "hot2"!
	for i := 0; i < 100; i++ {
		coldKey := fmt.Sprintf("cold_%d", i)
		cache.Set(coldKey, i)
	}

	// 3. Verify that TinyLFU admission filter preserved hot1 and hot2!
	_, ok1 := cache.Get("hot1")
	_, ok2 := cache.Get("hot2")

	if !ok1 && !ok2 {
		t.Fatalf("TinyLFU failed scan resistance test: hot items were evicted by cold scan")
	}
}

func TestMetricsHitRatio(t *testing.T) {
	cache, _ := tinylfu.New[int, string](10)
	defer cache.Close()

	cache.Set(1, "one")
	cache.Get(1) // hit
	cache.Get(2) // miss

	m := cache.Metrics()
	if m.Hits != 1 || m.Misses != 1 {
		t.Fatalf("expected 1 hit and 1 miss, got %d hits and %d misses", m.Hits, m.Misses)
	}
	if m.HitRatio != 0.5 {
		t.Fatalf("expected 0.5 hit ratio, got %f", m.HitRatio)
	}
}

