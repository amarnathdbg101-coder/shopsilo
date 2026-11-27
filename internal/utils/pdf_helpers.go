package utils

import (
	"strings"

	"github.com/signintech/gopdf"

	"shopMe/internal/handler/model"
)

// newDevanagariPDF initializes a gopdf document with the given page size, adds the first page,
// and registers the embedded Noto Sans Devanagari regular and bold fonts.
func newDevanagariPDF(pageSize gopdf.Rect) (*gopdf.GoPdf, error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: pageSize})
	pdf.AddPage()

	if err := SetupInvoiceFonts(pdf); err != nil {
		return nil, err
	}
	return pdf, nil
}

// pdfPalette holds the standard ShopSilo executive RGB color palette for native PDFs.
type pdfPalette struct {
	TopBanner []uint8
	HeaderSub []uint8
	Dark      []uint8
	Muted     []uint8
	TblHeader []uint8
	AltRow    []uint8
	Border    []uint8
	CardBg    []uint8
	Red       []uint8
	Green     []uint8
}

func executivePalette() pdfPalette {
	return pdfPalette{
		TopBanner: []uint8{15, 23, 42},    // #0f172a
		HeaderSub: []uint8{148, 163, 184}, // #94a3b8
		Dark:      []uint8{15, 23, 42},    // #0f172a
		Muted:     []uint8{71, 85, 105},   // #475569
		TblHeader: []uint8{49, 46, 129},   // #312e81
		AltRow:    []uint8{248, 250, 252}, // #f8fafc
		Border:    []uint8{226, 232, 240}, // #e2e8f0
		CardBg:    []uint8{248, 250, 252}, // #f8fafc
		Red:       []uint8{220, 38, 38},   // #dc2626
		Green:     []uint8{22, 163, 74},   // #16a34a
	}
}

type shopPDFMeta struct {
	Name    string
	Address string
	Phone   string
	UPI     string
}

func extractShopPDFMeta(shop *model.Shop, defaultName string) shopPDFMeta {
	meta := shopPDFMeta{Name: defaultName}
	if shop == nil {
		return meta
	}

	if trimmed := strings.TrimSpace(shop.Name); trimmed != "" {
		meta.Name = trimmed
	}

	var addrParts []string
	if addr := strings.TrimSpace(shop.Address); addr != "" {
		addrParts = append(addrParts, addr)
	}
	if city := strings.TrimSpace(shop.City); city != "" {
		if shop.Pincode != "" {
			city += " " + shop.Pincode
		}
		addrParts = append(addrParts, city)
	}
	meta.Address = strings.Join(addrParts, ", ")

	if shop.WhatsAppNumber != "" {
		meta.Phone = "Phone / WhatsApp: " + shop.WhatsAppNumber
	} else if shop.Phone != "" {
		meta.Phone = "Phone: " + shop.Phone
	}
	meta.UPI = strings.TrimSpace(shop.UPIID)
	return meta
}
