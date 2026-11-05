// Package utils provides helper utilities.
package utils

import (
	"strings"

	"shopMe/internal/handler/model"
)

func maskPhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 4 {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "+") && len(trimmed) >= 12 {
		return trimmed[:5] + "*****" + trimmed[len(trimmed)-3:]
	}
	if len(trimmed) >= 10 {
		return trimmed[:2] + "*****" + trimmed[len(trimmed)-3:]
	}
	return trimmed[:1] + "***" + trimmed[len(trimmed)-2:]
}

// GenerateKhataStatementPDF produces an executive-grade customer account passbook PDF using pure Go.
func GenerateKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	return GenerateNativeKhataStatementPDF(customer, transactions, shop)
}
