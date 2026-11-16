package gomoney

import (
	"strings"
	"sync"
)

// Currency describes an ISO-4217 currency specification.
type Currency struct {
	Code      string // e.g. "INR", "USD", "EUR", "JPY"
	Numeric   string // ISO 4217 numeric code e.g. "356", "840"
	Symbol    string // e.g. "₹", "$", "€", "¥"
	Scale     int    // decimal places (e.g. 2 for cents/paise, 0 for Yen, 3 for Dinar)
	Separator string // thousands grouping separator (e.g. ",")
	Decimal   string // decimal point separator (e.g. ".")
}

var (
	currMu    sync.RWMutex
	currsReg  = make(map[string]Currency)
)

// Predefined common currencies.
var (
	INR = Currency{Code: "INR", Numeric: "356", Symbol: "₹", Scale: 2, Separator: ",", Decimal: "."}
	USD = Currency{Code: "USD", Numeric: "840", Symbol: "$", Scale: 2, Separator: ",", Decimal: "."}
	EUR = Currency{Code: "EUR", Numeric: "978", Symbol: "€", Scale: 2, Separator: ".", Decimal: ","}
	GBP = Currency{Code: "GBP", Numeric: "826", Symbol: "£", Scale: 2, Separator: ",", Decimal: "."}
	JPY = Currency{Code: "JPY", Numeric: "392", Symbol: "¥", Scale: 0, Separator: ",", Decimal: "."}
	AED = Currency{Code: "AED", Numeric: "784", Symbol: "د.إ", Scale: 2, Separator: ",", Decimal: "."}
	KWD = Currency{Code: "KWD", Numeric: "414", Symbol: "د.ك", Scale: 3, Separator: ",", Decimal: "."}
	AUD = Currency{Code: "AUD", Numeric: "036", Symbol: "A$", Scale: 2, Separator: ",", Decimal: "."}
	CAD = Currency{Code: "CAD", Numeric: "124", Symbol: "C$", Scale: 2, Separator: ",", Decimal: "."}
	SGD = Currency{Code: "SGD", Numeric: "702", Symbol: "S$", Scale: 2, Separator: ",", Decimal: "."}
)

func init() {
	RegisterCurrency(INR)
	RegisterCurrency(USD)
	RegisterCurrency(EUR)
	RegisterCurrency(GBP)
	RegisterCurrency(JPY)
	RegisterCurrency(AED)
	RegisterCurrency(KWD)
	RegisterCurrency(AUD)
	RegisterCurrency(CAD)
	RegisterCurrency(SGD)
}

// RegisterCurrency adds or updates a currency in the registry.
func RegisterCurrency(c Currency) {
	currMu.Lock()
	defer currMu.Unlock()
	currsReg[strings.ToUpper(c.Code)] = c
}

// GetCurrency retrieves currency by its ISO code (case-insensitive).
func GetCurrency(code string) (Currency, bool) {
	currMu.RLock()
	defer currMu.RUnlock()
	c, found := currsReg[strings.ToUpper(code)]
	return c, found
}
