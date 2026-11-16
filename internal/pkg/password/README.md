# go-password 🔐

`go-password` ek enterprise-grade password security, validation, aur CSPRNG generation library hai Go ke liye. Ye Shannon information entropy calculate karta hai, OWASP aur NIST SP 800-63B standards ke mutabik passwords validate karta hai (common leaked passwords aur repeating characters reject karke), aur cryptographically secure random passwords generate karta hai.

---

## ✨ Features

- **Shannon Entropy Calculator**: Bit entropy calculate karke mathematical strength measure karein (`Entropy(pw)`).
- **5-Tier Strength Scoring**: `Very Weak`, `Weak`, `Medium`, `Strong`, `Very Strong`.
- **OWASP & NIST SP 800-63B Compliance**:
  - Embedded common leaked passwords blacklist.
  - Max consecutive repeating characters limit (`aaaa` rejected).
  - Minimum/Maximum length and character class rules.
- **CSPRNG Password Generator**: `crypto/rand` powered secure password creation with guaranteed character distribution.
- **Zero Third-Party Dependencies**: Pure Go Standard Library.

---

## 🚀 Quick Start

### Installation
```bash
go get github.com/shopsilo/go-password
```

### Complete Example
```go
package main

import (
	"fmt"

	"github.com/shopsilo/go-password"
)

func main() {
	// 1. Validate User Password during Registration
	policy := password.DefaultPolicy()

	err := password.Validate("password123", policy)
	if err != nil {
		fmt.Printf("Password rejected: %v\n", err) // ErrCommonPassword
	}

	// 2. Measure Entropy & Score
	userPw := "K#9xP$2wL!mQ"
	fmt.Printf("Entropy: %.2f bits | Strength: %s\n",
		password.Entropy(userPw), password.Score(userPw)) // ~78 bits | Very Strong

	// 3. Generate Secure Random Password for New User
	generated, err := password.Generate(password.GenerateOptions{
		Length:         16,
		IncludeUpper:   true,
		IncludeLower:   true,
		IncludeDigits:  true,
		IncludeSymbols: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated Secure Password: %s\n", generated)
}
```

---

## 🧪 Testing

```bash
go test -v -count=1 ./...
```
Output:
```
=== RUN   TestEntropyAndScore
--- PASS: TestEntropyAndScore (0.00s)
=== RUN   TestValidate_Policy
--- PASS: TestValidate_Policy (0.00s)
=== RUN   TestGenerate
--- PASS: TestGenerate (0.00s)
PASS
ok  	go-password	0.484s
```

---

## 📄 License
MIT License.
