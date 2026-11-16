# go-csvkit 📊

Memory-safe streaming CSV reader and writer with automated UTF-8 BOM stripping, bounded batch chunking, and type-safe row accessors. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🛑 **OOM-Proof Streaming Chunking**: 100MB+ ya millions of rows waali CSV files ko bina RAM bhare 1,000-row chunks me stream karta hai.
- 🪟 **Excel UTF-8 BOM Auto-Strip**: Windows Excel dwara lagaye gaye `\xef\xbb\xbf` BOM mark ko automatically clean karta hai taaki headers corrupt na hon.
- 🔢 **Typed Row Accessors**: `row.GetInt("qty")`, `row.GetFloat64("price")`, `row.GetBool("is_active")` bina manual `strconv` boilerplate ke.
- ✍️ **Map-Aligned Writer**: Map keys ko automatically headers ke column order ke anusaar map karke write karta hai.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Streaming Massive CSV in Chunks (Zero RAM Spike)
```go
package main

import (
	"fmt"
	"log"
	"os"
	gocsv "go-csvkit"
)

func main() {
	f, err := os.Open("massive_inventory.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	reader, err := gocsv.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}

	// Process in batches of 500 rows at a time
	err = reader.Chunk(500, func(rows []gocsv.Row) error {
		for _, row := range rows {
			sku := row.Get("sku")
			price := row.GetFloat64("price", 0.0)
			qty := row.GetInt("quantity", 0)
			_ = fmt.Sprintf("Processing %s: %d units @ ₹%.2f\n", sku, qty, price)
		}
		// Yahan DB me bulk insert / update query chala sakte hain!
		return nil
	})

	if err != nil {
		log.Fatal("Chunk processing error:", err)
	}
}
```

### 2. Writing CSV from Maps
```go
f, _ := os.Create("export.csv")
defer f.Close()

writer := gocsv.NewWriter(f)
_ = writer.WriteHeader("order_id", "customer_name", "amount")

_ = writer.WriteMap(map[string]string{
    "order_id":      "ORD-101",
    "customer_name": "Karan Malhotra",
    "amount":        "3499.00",
})

_ = writer.Flush()
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
