# go-slug 🐌

Ultra-fast, zero-dependency Unicode URL slugifier with native Hindi/Devanagari, Latin diacritics, and currency transliteration. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- ⚡ **Blazing Fast**: ~3.4 µs per operation, minimal memory allocations.
- 🇮🇳 **Devanagari & Hindi Support**: Seamlessly converts Hindi strings (`"भारत का नया स्मार्टफोन ₹14999"` -> `"bhart-ka-nya-smartphon-inr-14999"`).
- 🌍 **Latin Diacritics**: Automatically strips accents (`"Café au Lait"` -> `"cafe-au-lait"`).
- 🔣 **Symbols & Currencies**: Maps `&`, `@`, `₹`, `$`, `€`, `%` to readable words (`"and"`, `"at"`, `"inr"`, `"usd"`, etc.).
- ✂️ **Word-Boundary Truncation**: Truncates cleanly without chopping words in half.
- 📦 **Zero External Dependencies**: Pure Go Standard Library.

---

## Quick Start

### 1. Basic Slug Generation
```go
package main

import (
	"fmt"
	goslug "go-slug"
)

func main() {
	title := "Apple iPhone 16 Pro (256 GB) - ₹1,19,900!"
	slug := goslug.Make(title)
	// Output: apple-iphone-16-pro-256-gb-inr-1-19-900
	fmt.Println(slug)
}
```

### 2. Custom Configuration (Length Limit & Underscores)
```go
cfg := goslug.DefaultConfig
cfg.Separator = "_"
cfg.MaxLength = 30 // Truncates cleanly at word boundary

slug := goslug.MakeWithConfig("Super Fast E-Commerce Backend Architecture in Go", cfg)
// Output: super_fast_e_commerce_backend
fmt.Println(slug)
```

### 3. Validating Slugs
```go
if goslug.IsValidSlug("my-clean-slug-2026", "-") {
    fmt.Println("Slug is valid for SEO routing!")
}
```

---

## Running Tests & Benchmarks
```bash
go test -v -bench=. -benchmem ./...
```
