// Package utils provides helper utilities.
package utils

import "github.com/skip2/go-qrcode"

// GenerateQRCodePNG generates a PNG byte slice of a QR code for the given content.
func GenerateQRCodePNG(content string, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	return qrcode.Encode(content, qrcode.Medium, size)
}
