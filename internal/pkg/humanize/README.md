# go-humanize 👥

`go-humanize` ek lightweight, high-performance data humanization library hai Go ke liye. Ye raw integers, timestamps, aur counts ko readable, user-friendly language me format karta hai supporting international formats aur Indian numbering conventions (Lakhs & Crores).

---

## ✨ Features

- **Indian Numbering System**: `FormatIndian(1250000)` -> `12,50,000`, `FormatIndian(10000000)` -> `1,00,00,000`.
- **Western Comma Formatting**: `FormatComma(1250000)` -> `1,250,000`.
- **Compact Metric & Indian**:
  - `Compact(1500000)` -> `1.5M`.
  - `CompactIndian(15000000)` -> `1.5 Cr`.
- **Ordinal Numbers**: English ordinal suffix handler (`1st`, `2nd`, `3rd`, `11th`, `21st`).
- **Relative Time Formatting**: `TimeAgo(t)` -> `"just now"`, `"5 minutes ago"`, `"yesterday"`, `"3 months ago"`.
- **Pluralization**: Count-aware singular/plural text formatting (`Plural(1, "item", "items")` -> `"1 item"`).
- **Zero Third-Party Dependencies**: Pure Go Standard Library.

---

## 🚀 Quick Start

### Installation
```bash
go get github.com/shopsilo/go-humanize
```

### Complete Example
```go
package main

import (
	"fmt"
	"time"

	"github.com/shopsilo/go-humanize"
)

func main() {
	// 1. Indian Currency Formatting
	balance := int64(1250000)
	fmt.Printf("Account Balance: ₹%s\n", humanize.FormatIndian(balance)) // ₹12,50,000

	// 2. Compact Indian & Western Metrics
	fmt.Printf("Revenue: %s\n", humanize.CompactIndian(25000000)) // 2.5 Cr
	fmt.Printf("App Downloads: %s\n", humanize.Compact(1850000))  // 1.9M

	// 3. Ordinal Display
	rank := 3
	fmt.Printf("You finished in %s place!\n", humanize.Ordinal(rank)) // 3rd

	// 4. Relative Time
	orderPlaced := time.Now().Add(-45 * time.Minute)
	fmt.Printf("Order placed %s\n", humanize.TimeAgo(orderPlaced)) // 45 minutes ago

	// 5. Pluralization
	fmt.Println(humanize.Plural(1, "item", "items")) // 1 item
	fmt.Println(humanize.Plural(4, "item", "items")) // 4 items
}
```

---

## 🧪 Testing

```bash
go test -v -count=1 ./...
```
Output:
```
=== RUN   TestFormatComma
--- PASS: TestFormatComma (0.00s)
=== RUN   TestFormatIndian
--- PASS: TestFormatIndian (0.00s)
=== RUN   TestCompact
--- PASS: TestCompact (0.00s)
=== RUN   TestCompactIndian
--- PASS: TestCompactIndian (0.00s)
=== RUN   TestOrdinal
--- PASS: TestOrdinal (0.00s)
=== RUN   TestTimeAgo
--- PASS: TestTimeAgo (0.00s)
=== RUN   TestPlural
--- PASS: TestPlural (0.00s)
PASS
ok  	go-humanize	0.487s
```

---

## 📄 License
MIT License.
