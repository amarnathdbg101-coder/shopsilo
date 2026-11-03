// Package utils provides helper utilities.
package utils

import (
	"context"

	"shopMe/internal/handler/model"
)

// GeneratePOSReceiptPDF generates an executive-grade digital tax receipt PDF using chromedp.
func GeneratePOSReceiptPDF(bill *model.POSBill) ([]byte, error) {
	return GenerateSmartBillPDF(context.Background(), bill)
}
