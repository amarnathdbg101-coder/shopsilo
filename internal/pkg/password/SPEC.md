# Technical Specification: go-password

## 1. Domain & Purpose
`go-password` is an enterprise-grade password security, validation, and generation toolkit for Go. It calculates Shannon information entropy, validates passwords against OWASP and NIST SP 800-63B standards (disallowing dictionary attacks, common passwords, and consecutive repetitions), and produces cryptographically secure random passwords using `crypto/rand`.

## 2. Architecture & Design Principles
- **Shannon Information Entropy**: Computes exact bit entropy based on character pool diversity and length.
- **OWASP / NIST SP 800-63B Validation**: Enforces configurable minimum/maximum lengths, character class requirements, blacklist matching against top common leaked credentials, and consecutive character repeat ceilings.
- **CSPRNG Password Generation**: Guaranteed inclusion of selected character classes with Fisher-Yates randomization to eliminate position bias.
- **Zero Heavy Dependencies**: Pure standard library (`crypto/rand`, `math`, `strings`, `unicode`).

## 3. API Surface
- `type Strength int`
- `type Policy struct`
- `func DefaultPolicy() Policy`
- `func Entropy(pw string) float64`
- `func Score(pw string) Strength`
- `func Validate(pw string, p Policy) error`
- `type GenerateOptions struct`
- `func Generate(opts GenerateOptions) (string, error)`

## 4. Error Handling
- `ErrTooShort`: Password is shorter than required minimum.
- `ErrTooLong`: Password exceeds allowed maximum length.
- `ErrCommonPassword`: Password matches known breached or weak credential dictionary.
- `ErrRepeatingChars`: Password contains excessive consecutive identical runes.
- `ErrInsufficientEntropy`: Password entropy falls below policy requirement.

## 5. Verification & Testing
- Unit tests verify entropy boundaries, strength tier classifications, blacklist rejections, consecutive repeat detection, and cryptographically secure generation with balanced character classes.
