# SPEC: go-mask

## 1. Metadata
- **Module**: `go-mask`
- **Package**: `gomask`
- **Location**: `C:\Library\go-mask`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `strings`, `unicode`, `encoding/json`).

---

## 2. Core Capabilities
1. **Financial & Indian Regulatory PII Masking (DPDP Act & RBI Compliant)**:
   - `Aadhaar`: Masks 12-digit numbers into `XXXX-XXXX-1234`.
   - `PAN`: Masks 10-char alphanumeric IDs into `XXXXX1234X`.
   - `CreditCard`: Formats 16-digit cards into PCI-DSS compliant `4111-XXXX-XXXX-1234`.
2. **Contact Info Masking**:
   - `Email`: Masks username characters (`r***a@example.com`).
   - `Phone`: Preserves country code and last 4 digits (`+91-98****3210`).
3. **Recursive JSON Redaction**:
   - `JSONRecursively`: Traverses arbitrary JSON objects and arrays, masking sensitive keys (`password`, `token`, `secret`, `cvv`, `api_key`).
4. **Generic Boundary Masker**:
   - `MaskString(s, prefix, suffix, maskRune)` customizable rune masking.

---

## 3. Public API
```go
func Aadhaar(aadhaar string) string
func PAN(pan string) string
func CreditCard(cardNumber string) string
func Email(email string) string
func Phone(phone string) string
func MaskString(s string, keepPrefix, keepSuffix int, maskRune rune) string
func JSONRecursively(jsonBytes []byte, sensitiveKeys ...string) ([]byte, error)
```
