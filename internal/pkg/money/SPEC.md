# Technical Specification: go-money

## 1. Domain & Purpose
`go-money` is an ultra-precise, zero-float-drift monetary calculation and currency representation engine for Go. In e-commerce, banking, payment gateways, and accounting ledgers, floating-point arithmetic introduces IEEE-754 rounding inaccuracies. `go-money` prevents financial calculation anomalies by operating strictly on integer subunits (e.g. paise, cents) while providing ISO-4217 currency metadata, safe multi-party allocations, and SQL persistence.

## 2. Architecture & Design Principles
- **Integer Subunit Storage**: All amounts are represented internally as `int64` smallest units (e.g. 100 paise = ₹1.00).
- **Zero Financial Loss Division**: Standard division discards remainders. `go-money` implements banking allocation algorithms (`Split` and `Allocate`) that distribute rounding remainders penny-by-penny across parties, guaranteeing that `sum(parts) == total`.
- **Currency Isolation**: Strict type checks prevent accidental operations across disparate currencies (e.g. adding USD to INR returns `ErrCurrencyMismatch`).
- **Database & Serialization Interop**: Directly implements `driver.Valuer` and `sql.Scanner` for PostgreSQL `BIGINT`, alongside standard JSON marshaling.

## 3. API Surface
- `type Currency struct { Code, Numeric, Symbol string; Scale int; Separator, Decimal string }`
- `func RegisterCurrency(c Currency)`
- `func GetCurrency(code string) (Currency, bool)`
- `type Money struct { amount int64; currency Currency }`
- `func New(amount int64, currency Currency) Money`
- `func NewFromSubunits(amount int64, currencyCode string) (Money, error)`
- `func NewFromFloat(amount float64, currencyCode string) (Money, error)`
- `func NewFromString(amountStr string, currencyCode string) (Money, error)`
- `func (m Money) Add(other Money) (Money, error)`
- `func (m Money) Sub(other Money) (Money, error)`
- `func (m Money) Mul(factor int64) Money`
- `func (m Money) MulFloat(factor float64) Money`
- `func (m Money) Split(parts int) ([]Money, error)`
- `func (m Money) Allocate(ratios ...int) ([]Money, error)`
- `func (m Money) Format() string`
- `func (m Money) Float64() float64`

## 4. Error Handling
- `ErrCurrencyMismatch`: Binary arithmetic on non-matching currency codes.
- `ErrInvalidAmount`: Unparseable numeric strings.
- `ErrInvalidCurrency`: Unregistered or blank ISO code.
- `ErrInvalidSplit`: Non-positive partition count.
- `ErrInvalidAllocation`: Non-positive ratio weights.

## 5. Verification & Testing
- Unit tests verify integer subunit conversions, zero float drift, Split/Allocate zero-remainder-loss guarantees, currency formatting, JSON marshaling, and database/sql Scanner/Valuer drivers.
