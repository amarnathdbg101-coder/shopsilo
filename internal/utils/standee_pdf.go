// Package utils provides helper utilities.
package utils

import (
	"shopMe/internal/handler/model"
)

// GenerateStoreStandeePDF generates a professional, high-resolution printable A5 counter standee PDF in pure Go.
func GenerateStoreStandeePDF(shop *model.Shop) ([]byte, error) {
	return GenerateNativeStoreStandeePDF(shop)
}
