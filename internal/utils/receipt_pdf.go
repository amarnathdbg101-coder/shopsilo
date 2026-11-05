// Package utils provides helper utilities.
package utils

import (
	"shopMe/internal/handler/model"
)

// GeneratePOSReceiptPDF generates an executive-grade digital tax receipt PDF using pure native Go.
func GeneratePOSReceiptPDF(bill *model.POSBill) ([]byte, error) {
	return GenerateNativeBillPDF(bill, NativePDFOptions{Format: FormatA5})
}
