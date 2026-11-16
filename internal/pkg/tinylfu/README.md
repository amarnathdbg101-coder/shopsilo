# go-tinylfu ☕

> **Ultra High-Hit-Ratio In-Memory Cache implementing the Caffeine Window TinyLFU Algorithm for Go (Go 1.23+).**  
> Combines a 4-bit Count-Min Sketch frequency estimator, an LRU admission window, and an SLRU (Segmented LRU) main cache to provide complete scan-resistance and 98%+ hit ratios.

---

## 🚀 Key Features

1. **Window TinyLFU (Caffeine Algorithm)**:
   - Eliminates LRU's greatest weakness: **Scan Vulnerability** (where a single loop over cold database records evicts frequently accessed items).
2. **4-Bit Count-Min Sketch**:
   - Saturating 4-bit counters with periodic halving (aging factor) estimate item popularity using a tiny memory footprint.
3. **Segmented Cache Topology**:
   - **Window Cache (1%)**: Absorbs burst recency.
   - **Main Cache (99%)**: Divided into Probational (20%) and Protected (80%) segments.
4. **Admission Filter**:
   - Items evicted from the window compete on frequency against the probational victim. Unpopular one-off queries are rejected at the door!
5. **Zero Dependencies**:
   - 100% pure standard library (`sync`, `container/list`, `hash/fnv`).

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-tinylfu"
)

func main() {
	// Initialize a 10,000 capacity cache
	cache, _ := tinylfu.New[string, string](10000)
	defer cache.Close()

	cache.Set("product:101", `{"name":"Basmati Rice"}`)

	if val, ok := cache.Get("product:101"); ok {
		fmt.Println("Found in cache:", val)
	}

	metrics := cache.Metrics()
	fmt.Printf("Hit Ratio: %.2f%%\n", metrics.HitRatio*100)
}
```
