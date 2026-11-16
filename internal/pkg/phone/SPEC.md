# Technical Specification: go-phone

## 1. Domain & Purpose
`go-phone` is a fast, lightweight telephone number parsing, normalization, and validation library for Go. It standardizes phone numbers into international E.164 format, handles national trunk prefixes, validates Indian 10-digit mobile numbers, and provides privacy masking for audit logs and UI displays without heavy external C++ dependencies.

## 2. Architecture & Design Principles
- **E.164 Standardization**: Parses numbers with international `+` prefixes or applies configurable fallback calling codes.
- **Smart Trunk Prefix Handling**: Automatically handles domestic leading trunk zeroes (e.g., `09876543210` -> `+919876543210`).
- **Indian Telecom Compliance**: Built-in validation for 10-digit Indian cellular numbers (starting with 6, 7, 8, or 9) compliant with DoT/TRAI numbering plans.
- **Privacy Masking**: Masks sensitive middle digits while keeping country code and last 4 digits visible for authentication screens.
- **Zero Heavy Dependencies**: Pure standard library (`strings`, `unicode`, `fmt`, `sort`).

## 3. API Surface
- `type PhoneNumber struct`
- `func Parse(raw string, defaultCountryCode string) (*PhoneNumber, error)`
- `func FormatE164(raw string, defaultCountryCode string) (string, error)`
- `func CleanDigits(raw string) string`
- `func IsValidIndianMobile(raw string) bool`
- `func Mask(raw string, defaultCountryCode string) string`

## 4. Error Handling
- `ErrEmptyPhone`: Input string contains no digits.
- `ErrInvalidLength`: Normalized digit length does not comply with E.164 limits (7 to 15 digits).
- `ErrInvalidCountryCode`: Number lacks international `+` prefix and no default calling code was provided.
- `ErrInvalidCharacters`: Input contains disallowed character sequences.

## 5. Verification & Testing
- Unit tests verify international code recognition, domestic trunk stripping, Indian mobile rules, privacy masking, direct E.164 formatting, and error scenarios.
