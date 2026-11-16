# go-money

Zero-float-drift, high-precision monetary math, ISO-4217 currencies, and safe financial allocations for Go.

---

## Why go-money?
In Go, using standard `float64` for money leads to precision corruption:
```go
// Standard float: 0.10 + 0.20 = 0.30000000000000004 ❌ (Loses money over time)
```
`go-money` stores amounts safely as `int64` smallest currency subunits (paise, cents, yen) and provides zero-loss partitioning so your ledgers and invoices reconcile to the exact cent:
```go
// go-money: ₹100 split 3 ways = ₹33.34 + ₹33.33 + ₹33.33 == ₹100.00 exactly! ✅
```

---

## Features
- **Zero Rounding Drift**: Operates strictly on integer subunits with banker's rounding on fractional multiplications.
- **Fair Banking Allocation**: `Split(parts)` and `Allocate(ratios...)` distribute remainder pennies sequentially so no money vanishes.
- **Multi-Currency Safety**: Type-checked currency matching prevents accidental additions across currencies (e.g. `INR + USD` produces an error).
- **Currencies Included**: Out-of-the-box support for `INR`, `USD`, `EUR`, `GBP`, `JPY`, `AED`, `AUD`, `CAD`, and extensible via `RegisterCurrency`.
- **Database & JSON Ready**: Implements `database/sql/driver.Valuer`, `sql.Scanner`, and standard JSON marshaling.

---

## Quickstart

```go
package main

import (
	"fmt"

	gomoney "go-money"
)

func main() {
	// 1. Create monetary amount (₹1,499.50)
	price, err := gomoney.NewFromFloat(1499.50, "INR")
	if err != nil {
		panic(err)
	}

	// 2. Add 18% GST (tax = price * 0.18)
	tax := price.MulFloat(0.18)
	total, _ := price.Add(tax)

	fmt.Printf("Base: %s\n", price.Format()) // ₹1,499.50
	fmt.Printf("Tax:  %s\n", tax.Format())   // ₹269.91
	fmt.Printf("Total: %s\n", total.Format()) // ₹1,769.41

	// 3. Fair Split without cent loss (e.g. 3 roommates dividing bill)
	shares, _ := total.Split(3)
	for i, share := range shares {
		fmt.Printf("Person %d share: %s\n", i+1, share.Format())
	}
	// Person 1: ₹589.81
	// Person 2: ₹589.80
	// Person 3: ₹589.80 (Sum is exactly ₹1,769.41)
}
```

---

## Testing
Run unit tests with full race and count coverage:
```bash
go test -v -count=1 ./...
```
