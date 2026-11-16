# go-imghash 🖼️

Pure Go Perceptual Image Difference Hashing (dHash) and Similarity Matcher.

## 🚀 Key Features
- **Zero External Dependencies**: Pure Go Standard Library (`image`, `math/bits`).
- **Aspect-Ratio & Resizing Invariant**: Correctly matches images even after resizing or format transcoding.
- **Fast Bitwise Comparison**: Uses `math/bits.OnesCount64` for sub-microsecond Hamming distance calculations.
- **Deduplication Ready**: Ideal for e-commerce catalog image deduplication and reverse image matching.

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"os"

	"github.com/shopsilo/go-imghash"
)

func main() {
	f1, _ := os.Open("product_original.jpg")
	defer f1.Close()

	f2, _ := os.Open("product_thumbnail.png")
	defer f2.Close()

	// Compute 64-bit dHash
	hash1, _ := imghash.Compute(f1)
	hash2, _ := imghash.Compute(f2)

	// Compare similarity (Default distance threshold <= 5)
	if imghash.Similar(hash1, hash2) {
		dist, _ := imghash.Distance(hash1, hash2)
		fmt.Printf("Images match! Hamming distance: %d\n", dist)
	}
}
```
