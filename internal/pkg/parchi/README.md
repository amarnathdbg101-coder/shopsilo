# go-parchi

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-parchi.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-parchi)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Universal, deterministic Kirana & grocery slip parser for Indian retail POS, quick-commerce, and WhatsApp ordering workflows in pure Go.

Zero external dependencies. Sub-microsecond execution (~6.6 μs/op). Saves 100% of LLM token costs for structured and semi-structured slips.

---

## Features
- **Offline & Zero-Token Fast Path**: Decodes grocery parchi and counter voice orders locally without pinging external AI APIs.
- **Bilingual & Colloquial Unit Normalization**: Standardizes 40+ Indian retail units across Hindi, Hinglish, and English (`kilo` -> `kg`, `pkt` -> `packet`, `ltr` -> `litre`, `bora` -> `sack`, `darjan` -> `dozen`, `dabba` -> `box`, `botal` -> `bottle`).
- **Flexible Pattern Matching**:
  - **Price-First**: `"50 ka tel"`, `"100 rupaye ki cheeni"`
  - **Qty-First**: `"2 kg cheeni"`, `"1 packet amul butter"`, `"3 litre tel 150 rs"`
  - **Name-First**: `"cheeni 2 kg"`, `"amul butter 1 packet 55 rs"`
- **List & Bullet Cleansing**: Automatically handles `"1. "`, `"2) "`, `"- "`, `"* "` bullets in multi-line WhatsApp pastes.
- **Conversational Detector**: Automatically detects open-ended questions (`"bhaiya sasta kaunsa hai"`, `"discount milega kya"`) and yields `ok=false` for LLM delegation.

---

## Installation
```bash
go get github.com/amarnathdbg101-coder/go/go-parchi@v1.0.0
```

---

## Quick Start

### 1. Parse a Single Item Line
```go
package main

import (
    "fmt"
    "github.com/amarnathdbg101-coder/go/go-parchi"
)

func main() {
    item, ok := parchi.ParseSingleLine("2 kg cheeni 80 rs")
    if ok {
        fmt.Printf("%s: %.1f %s @ ₹%.2f = ₹%.2f\n", 
            item.ProductName, item.Quantity, item.Unit, item.UnitPrice, item.TotalPrice)
        // Output: Cheeni: 2.0 kg @ ₹40.00 = ₹80.00
    }
}
```

### 2. Parse a Full WhatsApp / Counter Slip
```go
rawSlip := `1. 2 kg cheeni 80 rs
2) 1 packet amul butter 55 rs
* 50 ka tel`

slip, ok := parchi.ParseSlip(rawSlip)
if ok {
    fmt.Printf("Parsed %d items with total ₹%.2f\n", slip.MatchedCount, slip.EstimatedTotal)
} else {
    fmt.Printf("Partial match: %d matched, %d need AI assistance\n", slip.MatchedCount, slip.UnmatchedCount)
}
```

---

## Benchmarks
```
BenchmarkParseSingleLine-8    175562     6685 ns/op     322 B/op     9 allocs/op
BenchmarkParseSlip-8           84148    14588 ns/op    1661 B/op    35 allocs/op
BenchmarkSplitCommands-8      375822     3136 ns/op     597 B/op     8 allocs/op
```

---

## License
MIT
