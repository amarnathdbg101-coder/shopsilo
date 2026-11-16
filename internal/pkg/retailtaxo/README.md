# go-retailtaxo

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-retailtaxo.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-retailtaxo)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Universal Indian retail taxonomy, bilingual classification, and typo-tolerant category normalization engine in pure Go.

Includes 21 standardized retail categories, 100+ colloquial aliases, Hindi/English translations, and multi-tier fuzzy matching.

---

## Features
- **21 Canonical Retail Categories**: Standardized across Grocery, Dairy, Personal Care, Fashion, Home & Kitchen, Pharmacy, Hardware, and Electronics.
- **Bilingual & Multi-Store**: Built-in English and Hindi names (`किराना व राशन`, `डेयरी व नाश्ता`), departmental groupings, and visual emojis.
- **3-Tier Fuzzy Normalizer**:
  - Exact slug/name hash lookup (~1.2 μs/op).
  - Punctuation & hyphen agnostic matching.
  - Typo-tolerant phonetic & string distance matching (`"kirna"` -> `"kirana-grocery"`, `"snaks"` -> `"snacks-beverages"`).
- **Intelligent Title Formatter**: Formats strings like `"oil/ghee"` into `"Oil/Ghee"` and `"kirana (ration)"` into `"Kirana (Ration)"` in ~497 ns.

---

## Installation
```bash
go get github.com/amarnathdbg101-coder/go/go-retailtaxo@v1.0.0
```

---

## Quick Start

### 1. Resolve Slugs and Aliases
```go
package main

import (
    "fmt"
    "github.com/amarnathdbg101-coder/go/go-retailtaxo"
)

func main() {
    // Exact or synonym match
    slug, ok := retailtaxo.ResolveSlug("Dairy, Bakery & Frozen")
    fmt.Println(slug, ok) // "dairy-breakfast", true

    // Typo-tolerant fuzzy match
    slug, ok = retailtaxo.ResolveSlug("vegitables")
    fmt.Println(slug, ok) // "fruits-vegetables", true
}
```

### 2. Format Category Display Titles
```go
title := retailtaxo.FormatTitle("dry fruits & nuts")
fmt.Println(title) // "Dry Fruits & Nuts"
```

---

## Benchmarks
```
BenchmarkResolveSlug_Exact-8    918322     1276 ns/op     145 B/op     7 allocs/op
BenchmarkFormatTitle-8         2483574      497 ns/op     144 B/op     3 allocs/op
BenchmarkResolveSlug_Fuzzy-8      6878   173185 ns/op   37657 B/op   319 allocs/op
```

---

## License
MIT
