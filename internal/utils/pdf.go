// Package utils provides helper utilities.
package utils

import (
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

// GenerateWholesaleReorderPDF generates a ready-to-print PDF purchase order sheet for wholesale suppliers in pure Go.
func GenerateWholesaleReorderPDF(shop *model.Shop, items []*dto.LowStockProduct) ([]byte, error) {
	return GenerateNativeWholesaleReorderPDF(shop, items)
}

// GenerateCustomProcurementPDF generates a custom procurement list PDF in pure Go.
func GenerateCustomProcurementPDF(shop *model.Shop, title string, items []dto.ProcurementPDFItem) ([]byte, error) {
	return GenerateNativeCustomProcurementPDF(shop, title, items)
}
