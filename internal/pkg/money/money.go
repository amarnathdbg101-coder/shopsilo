package gomoney

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Money represents a monetary amount stored safely as integer subunits (cents, paise).
type Money struct {
	amount   int64
	currency Currency
}

// New creates a Money instance using subunits (e.g. 1000 subunits in USD is $10.00).
func New(amount int64, currency Currency) Money {
	return Money{
		amount:   amount,
		currency: currency,
	}
}

// NewFromSubunits creates a Money instance from an ISO currency code.
func NewFromSubunits(amount int64, currencyCode string) (Money, error) {
	curr, ok := GetCurrency(currencyCode)
	if !ok {
		return Money{}, fmt.Errorf("%w: %s", ErrInvalidCurrency, currencyCode)
	}
	return New(amount, curr), nil
}

// NewFromFloat creates a Money instance by converting a standard decimal float
// into integer subunits using proper half-away-from-zero rounding.
func NewFromFloat(amount float64, currencyCode string) (Money, error) {
	curr, ok := GetCurrency(currencyCode)
	if !ok {
		return Money{}, fmt.Errorf("%w: %s", ErrInvalidCurrency, currencyCode)
	}
	scaleMultiplier := math.Pow10(curr.Scale)
	subunits := int64(math.Round(amount * scaleMultiplier))
	return New(subunits, curr), nil
}

// NewFromString parses standard numeric string (e.g. "1499.50") into Money.
func NewFromString(amountStr string, currencyCode string) (Money, error) {
	amountStr = strings.TrimSpace(amountStr)
	val, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %s", ErrInvalidAmount, amountStr)
	}
	return NewFromFloat(val, currencyCode)
}

// Amount returns the raw subunits value (e.g. 149950 paise).
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the currency metadata.
func (m Money) Currency() Currency {
	return m.currency
}

// CurrencyCode returns ISO-4217 currency code.
func (m Money) CurrencyCode() string {
	return m.currency.Code
}

// Float64 converts the integer subunits to a decimal float64 representation.
func (m Money) Float64() float64 {
	if m.currency.Scale == 0 {
		return float64(m.amount)
	}
	return float64(m.amount) / math.Pow10(m.currency.Scale)
}

// Add adds another Money amount of identical currency.
func (m Money) Add(other Money) (Money, error) {
	if m.currency.Code != other.currency.Code {
		return Money{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency.Code, other.currency.Code)
	}
	return New(m.amount+other.amount, m.currency), nil
}

// Sub subtracts another Money amount of identical currency.
func (m Money) Sub(other Money) (Money, error) {
	if m.currency.Code != other.currency.Code {
		return Money{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency.Code, other.currency.Code)
	}
	return New(m.amount-other.amount, m.currency), nil
}

// Mul multiplies the monetary amount by an integer factor.
func (m Money) Mul(factor int64) Money {
	return New(m.amount*factor, m.currency)
}

// MulFloat multiplies by a floating point factor with exact half-away-from-zero rounding.
func (m Money) MulFloat(factor float64) Money {
	rounded := int64(math.Round(float64(m.amount) * factor))
	return New(rounded, m.currency)
}

// Abs returns absolute value of the monetary amount.
func (m Money) Abs() Money {
	if m.amount < 0 {
		return New(-m.amount, m.currency)
	}
	return m
}

// Negate returns negative value of the monetary amount.
func (m Money) Negate() Money {
	return New(-m.amount, m.currency)
}

// Split divides the amount into n parts without losing any sub-penny/sub-paise remainder.
// Remainder is distributed sequentially 1 subunit at a time.
func (m Money) Split(parts int) ([]Money, error) {
	if parts <= 0 {
		return nil, ErrInvalidSplit
	}

	base := m.amount / int64(parts)
	remainder := m.amount % int64(parts)

	if remainder < 0 {
		remainder = -remainder
	}

	result := make([]Money, parts)
	for i := 0; i < parts; i++ {
		share := base
		if int64(i) < remainder {
			if m.amount >= 0 {
				share++
			} else {
				share--
			}
		}
		result[i] = New(share, m.currency)
	}

	return result, nil
}

// Allocate divides the amount proportionally based on weights/ratios without rounding loss.
func (m Money) Allocate(ratios ...int) ([]Money, error) {
	if len(ratios) == 0 {
		return nil, ErrInvalidAllocation
	}

	totalWeight := int64(0)
	for _, r := range ratios {
		if r < 0 {
			return nil, ErrInvalidAllocation
		}
		totalWeight += int64(r)
	}
	if totalWeight <= 0 {
		return nil, ErrInvalidAllocation
	}

	result := make([]Money, len(ratios))
	remainder := m.amount

	for i, r := range ratios {
		share := (m.amount * int64(r)) / totalWeight
		result[i] = New(share, m.currency)
		remainder -= share
	}

	// Distribute remainder 1 subunit at a time
	step := int64(1)
	if remainder < 0 {
		step = -1
	}

	idx := 0
	for remainder != 0 {
		result[idx] = New(result[idx].amount+step, m.currency)
		remainder -= step
		idx = (idx + 1) % len(ratios)
	}

	return result, nil
}

// Equal returns true if currency and subunit amount are identical.
func (m Money) Equal(other Money) bool {
	return m.currency.Code == other.currency.Code && m.amount == other.amount
}

// GreaterThan checks if m > other. Returns error if currencies differ.
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.currency.Code != other.currency.Code {
		return false, ErrCurrencyMismatch
	}
	return m.amount > other.amount, nil
}

// LessThan checks if m < other. Returns error if currencies differ.
func (m Money) LessThan(other Money) (bool, error) {
	if m.currency.Code != other.currency.Code {
		return false, ErrCurrencyMismatch
	}
	return m.amount < other.amount, nil
}

// IsZero returns true if amount is 0.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive returns true if amount > 0.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if amount < 0.
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// Format formats the monetary amount into human-readable representation with symbol.
// E.g. "₹1,499.50", "$10.00", "¥500"
func (m Money) Format() string {
	absAmount := m.amount
	sign := ""
	if absAmount < 0 {
		sign = "-"
		absAmount = -absAmount
	}

	if m.currency.Scale == 0 {
		wholeFormatted := formatGrouping(strconv.FormatInt(absAmount, 10), m.currency.Separator)
		return fmt.Sprintf("%s%s%s", sign, m.currency.Symbol, wholeFormatted)
	}

	scaleFactor := int64(math.Pow10(m.currency.Scale))
	whole := absAmount / scaleFactor
	fraction := absAmount % scaleFactor

	wholeFormatted := formatGrouping(strconv.FormatInt(whole, 10), m.currency.Separator)
	fracFormat := fmt.Sprintf("%%0%dd", m.currency.Scale)
	fracFormatted := fmt.Sprintf(fracFormat, fraction)

	return fmt.Sprintf("%s%s%s%s%s", sign, m.currency.Symbol, wholeFormatted, m.currency.Decimal, fracFormatted)
}

func formatGrouping(s string, sep string) string {
	if sep == "" || len(s) <= 3 {
		return s
	}

	var parts []string
	n := len(s)
	for n > 3 {
		parts = append([]string{s[n-3 : n]}, parts...)
		n -= 3
	}
	if n > 0 {
		parts = append([]string{s[0:n]}, parts...)
	}

	return strings.Join(parts, sep)
}

// String implements fmt.Stringer.
func (m Money) String() string {
	return m.Format()
}

// JSON representation for API payloads.
type jsonMoney struct {
	Amount    int64   `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted"`
	Float     float64 `json:"float"`
}

// MarshalJSON serializes Money safely.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMoney{
		Amount:    m.amount,
		Currency:  m.currency.Code,
		Formatted: m.Format(),
		Float:     m.Float64(),
	})
}

// UnmarshalJSON deserializes JSON into Money.
func (m *Money) UnmarshalJSON(data []byte) error {
	var jm jsonMoney
	if err := json.Unmarshal(data, &jm); err != nil {
		return err
	}

	curr, ok := GetCurrency(jm.Currency)
	if !ok {
		return fmt.Errorf("%w: %s", ErrInvalidCurrency, jm.Currency)
	}

	m.amount = jm.Amount
	m.currency = curr
	return nil
}

// Value implements database/sql/driver.Valuer for PostgreSQL BIGINT persistence.
func (m Money) Value() (driver.Value, error) {
	return m.amount, nil
}

// Scan implements database/sql.Scanner for loading from DB.
func (m *Money) Scan(src any) error {
	if src == nil {
		m.amount = 0
		return nil
	}

	switch v := src.(type) {
	case int64:
		m.amount = v
	case int32:
		m.amount = int64(v)
	case int:
		m.amount = int64(v)
	case []byte:
		n, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return err
		}
		m.amount = n
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		m.amount = n
	default:
		return fmt.Errorf("gomoney: cannot scan type %T into Money", src)
	}

	if m.currency.Code == "" {
		m.currency = INR // Default fallback
	}
	return nil
}
