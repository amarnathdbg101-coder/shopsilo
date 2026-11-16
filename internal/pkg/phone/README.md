# go-phone 📞

`go-phone` ek lightweight, zero-dependency international telephone number parser, normalizer, aur Indian mobile validator library hai Go ke liye. Ye user-provided phone numbers ko clean karke standard E.164 format (`+919876543210`) me convert karta hai bina kisi heavy C++ bindings ya bloated metadata database ke.

---

## ✨ Features

- **E.164 International Normalization**: Automatic formatting like `+91 98765-43210` -> `+919876543210`.
- **Trunk Prefix Stripping**: Domestic prefixes jaise `09876543210` ko safely handle karta hai.
- **Indian Mobile Number Validator**: DoT/TRAI rules ke anusaar 10-digit Indian mobiles (starting with 6, 7, 8, 9) ko validate karta hai (`IsValidIndianMobile`).
- **Privacy Masking**: SMS/OTP and profile screens ke liye privacy-safe masking (`+91 ******3210`).
- **Clean Digit Extraction**: Extra spaces, dashes, brackets ko sub-microsecond time me strip karta hai.
- **Zero Third-Party Dependencies**: Pure Go Standard Library.

---

## 🚀 Quick Start

### Installation
```bash
go get github.com/shopsilo/go-phone
```

### Basic Parsing & E.164 Formatting
```go
package main

import (
	"fmt"
	"log"

	"github.com/shopsilo/go-phone"
)

func main() {
	// Parse with default fallback country code (+91)
	pn, err := phone.Parse("98765-43210", "+91")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Country Code:", pn.CountryCode)     // +91
	fmt.Println("National:", pn.NationalNumber)      // 9876543210
	fmt.Println("E.164:", pn.E164)                   // +919876543210

	// Validate Indian mobile
	if phone.IsValidIndianMobile(pn.Raw) {
		fmt.Println("Valid Indian cellular number!")
	}

	// Mask for display
	fmt.Println("Masked:", phone.Mask(pn.Raw, "+91")) // +91 ******3210
}
```

---

## 🧪 Testing

```bash
go test -v -count=1 ./...
```
Output:
```
=== RUN   TestParse_InternationalFormat
--- PASS: TestParse_InternationalFormat (0.00s)
=== RUN   TestParse_DefaultCountryCodeAndTrunkPrefix
--- PASS: TestParse_DefaultCountryCodeAndTrunkPrefix (0.00s)
=== RUN   TestIsValidIndianMobile
--- PASS: TestIsValidIndianMobile (0.00s)
=== RUN   TestMask
--- PASS: TestMask (0.00s)
=== RUN   TestFormatE164
--- PASS: TestFormatE164 (0.00s)
=== RUN   TestErrors
--- PASS: TestErrors (0.00s)
PASS
ok  	go-phone	0.472s
```

---

## 📄 License
MIT License.
