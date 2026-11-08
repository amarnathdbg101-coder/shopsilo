// Package utils provides helper utilities.
package utils

import (
	"shopMe/internal/handler/model"

	gomask "github.com/amarnathdbg101-coder/go/go-mask"
)

func maskPhone(phone string) string {
	return gomask.Phone(phone)
}

// GenerateKhataStatementPDF produces an executive-grade customer account passbook PDF using pure Go.
func GenerateKhataStatementPDF(customer *model.CustomerKhata, transactions []model.KhataTransaction, shop *model.Shop) ([]byte, error) {
	return GenerateNativeKhataStatementPDF(customer, transactions, shop)
}
