package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Calculate EAN-13 checksum digit
func ean13Checksum(code12 string) int {
	sum := 0
	for i, r := range code12 {
		digit := int(r - '0')
		if i%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	rem := sum % 10
	if rem == 0 {
		return 0
	}
	return 10 - rem
}

func generateValidEAN13(index int) string {
	// 890 is India GS1 country prefix + 8-digit unique index + 1 check digit
	base := fmt.Sprintf("890%08d", index%100000000)
	checksum := ean13Checksum(base)
	return fmt.Sprintf("%s%d", base, checksum)
}

func extractUnit(name string) string {
	lower := strings.ToLower(name)
	re := regexp.MustCompile(`(\d+(\.\d+)?)\s*(kg|g|gm|gram|grams|l|lt|liter|litres|litre|ml|pcs|piece|pieces|capsule|capsules|pack|sachet)`)
	m := re.FindString(lower)
	if m != "" {
		return m
	}
	if strings.Contains(lower, "bottle") {
		return "1 bottle"
	}
	if strings.Contains(lower, "bar") || strings.Contains(lower, "soap") {
		return "1 bar"
	}
	if strings.Contains(lower, "pack") || strings.Contains(lower, "pouch") {
		return "1 pack"
	}
	return "piece"
}

func getCategoryAttrs(cat, subCat string) (hsn string, gst float64, imgUrl string) {
	cLow := strings.ToLower(cat)
	sLow := strings.ToLower(subCat)

	// Defaults
	hsn = "19053100"
	gst = 18.0
	imgUrl = "https://images.unsplash.com/photo-1558961363-fa8fdf82db35?auto=format&fit=crop&w=500&q=80" // general packaged item

	if strings.Contains(cLow, "beauty") || strings.Contains(sLow, "hair") || strings.Contains(sLow, "bath") || strings.Contains(sLow, "skin") {
		hsn = "33051090"
		gst = 18.0
		imgUrl = "https://images.unsplash.com/photo-1535585209827-a15fcdbc4c2d?auto=format&fit=crop&w=500&q=80" // shampoo / personal care
		if strings.Contains(sLow, "soap") || strings.Contains(sLow, "bath") {
			hsn = "34011110"
			imgUrl = "https://images.unsplash.com/photo-1607006314187-c793ff615418?auto=format&fit=crop&w=500&q=80" // soap bar
		} else if strings.Contains(sLow, "oral") || strings.Contains(sLow, "toothpaste") {
			hsn = "33061020"
			imgUrl = "https://images.unsplash.com/photo-1559591937-e1022830f3a0?auto=format&fit=crop&w=500&q=80" // oral care
		}
	} else if strings.Contains(cLow, "clean") || strings.Contains(sLow, "detergent") || strings.Contains(sLow, "cleaner") {
		hsn = "34029099"
		gst = 18.0
		imgUrl = "https://images.unsplash.com/photo-1583947215259-38e31be8751f?auto=format&fit=crop&w=500&q=80" // detergent
		if strings.Contains(sLow, "dishwash") || strings.Contains(sLow, "scrub") {
			imgUrl = "https://images.unsplash.com/photo-1585421514738-01798e348b17?auto=format&fit=crop&w=500&q=80" // dishwash
		}
	} else if strings.Contains(sLow, "oil") || strings.Contains(sLow, "ghee") {
		hsn = "15121910"
		gst = 5.0
		imgUrl = "https://images.unsplash.com/photo-1474979266404-7eaacbcd87c5?auto=format&fit=crop&w=500&q=80" // oil
	} else if strings.Contains(sLow, "tea") || strings.Contains(sLow, "coffee") || strings.Contains(sLow, "beverage") || strings.Contains(cLow, "beverage") {
		hsn = "09024020"
		gst = 5.0
		imgUrl = "https://images.unsplash.com/photo-1576092768241-dec231879fc3?auto=format&fit=crop&w=500&q=80" // tea
		if strings.Contains(sLow, "coffee") {
			hsn = "21011110"
			gst = 18.0
			imgUrl = "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?auto=format&fit=crop&w=500&q=80" // coffee
		}
	} else if strings.Contains(sLow, "dairy") || strings.Contains(sLow, "milk") || strings.Contains(sLow, "butter") || strings.Contains(sLow, "cheese") {
		hsn = "04051000"
		gst = 12.0
		imgUrl = "https://images.unsplash.com/photo-1589985270826-4b7bb135bc9d?auto=format&fit=crop&w=500&q=80" // butter / dairy
	} else if strings.Contains(sLow, "chocolate") || strings.Contains(sLow, "sweet") {
		hsn = "18063200"
		gst = 18.0
		imgUrl = "https://images.unsplash.com/photo-1541781774459-bb2af2f05b55?auto=format&fit=crop&w=500&q=80" // chocolate
	} else if strings.Contains(sLow, "noodle") || strings.Contains(sLow, "pasta") {
		hsn = "19023010"
		gst = 12.0
		imgUrl = "https://images.unsplash.com/photo-1612927601601-6638404737ce?auto=format&fit=crop&w=500&q=80" // instant noodles
	} else if strings.Contains(sLow, "salt") || strings.Contains(sLow, "spice") || strings.Contains(sLow, "masala") {
		hsn = "25010010"
		gst = 5.0
		imgUrl = "https://images.unsplash.com/photo-1518977676601-b53f82aba655?auto=format&fit=crop&w=500&q=80" // salt / spices
	} else if strings.Contains(sLow, "atta") || strings.Contains(sLow, "flour") || strings.Contains(sLow, "rice") || strings.Contains(sLow, "dal") {
		hsn = "11010000"
		gst = 0.0
		imgUrl = "https://images.unsplash.com/photo-1586201375761-83865001e31c?auto=format&fit=crop&w=500&q=80" // grains / flour
	}

	return
}

func main() {
	inPath := "data/bigbasket_raw.csv"
	seedPath := "data/master_catalog_mega.csv"
	outPath := "data/master_catalog_mega_thousands.csv"

	fmt.Println("🚀 Converting and compiling thousands of authentic Indian FMCG products...")

	seenBarcodes := make(map[string]bool)
	seenNames := make(map[string]bool)
	var allRows [][]string

	header := []string{
		"barcode", "name", "brand", "category", "unit",
		"mrp", "selling_price", "image_url", "hsn", "gst", "description",
	}

	// 1. Seed already verified products first (Parle-G, KitKat, Maggi, Surf Excel etc.)
	seedF, err := os.Open(seedPath)
	if err == nil {
		r := csv.NewReader(seedF)
		r.FieldsPerRecord = -1
		_, _ = r.Read() // skip header
		for {
			rec, err := r.Read()
			if err != nil {
				break
			}
			if len(rec) >= 2 {
				bc := strings.TrimSpace(rec[0])
				nm := strings.ToLower(strings.TrimSpace(rec[1]))
				if bc != "" && nm != "" && !seenBarcodes[bc] {
					seenBarcodes[bc] = true
					seenNames[nm] = true
					allRows = append(allRows, rec)
				}
			}
		}
		seedF.Close()
		fmt.Printf("✅ Preserved %d core verified products\n", len(allRows))
	}

	// 2. Parse and convert BigBasket 40k products
	bbF, err := os.Open(inPath)
	if err != nil {
		fmt.Printf("❌ Failed to open %s: %v\n", inPath, err)
		return
	}
	defer bbF.Close()

	r := csv.NewReader(bbF)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	_, _ = r.Read() // skip header (index,product,category,sub_category,brand,sale_price,market_price,type,rating,description)

	idx := 1000 // barcode counter offset
	addedCount := 0

	// Target 5,000 to 10,000 clean unique products
	maxProducts := 10000

	for {
		rec, err := r.Read()
		if err == io.EOF || len(allRows) >= maxProducts {
			break
		}
		if err != nil || len(rec) < 7 {
			continue
		}

		name := strings.TrimSpace(rec[1])
		cat := strings.TrimSpace(rec[2])
		subCat := strings.TrimSpace(rec[3])
		brand := strings.TrimSpace(rec[4])
		salePriceStr := strings.TrimSpace(rec[5])
		mrpStr := strings.TrimSpace(rec[6])
		desc := ""
		if len(rec) > 9 {
			desc = strings.TrimSpace(rec[9])
		}

		if name == "" || brand == "" {
			continue
		}

		nameLower := strings.ToLower(name)
		if seenNames[nameLower] {
			continue
		}

		salePrice, _ := strconv.ParseFloat(salePriceStr, 64)
		mrp, _ := strconv.ParseFloat(mrpStr, 64)
		if mrp <= 0 && salePrice > 0 {
			mrp = salePrice
		}
		if salePrice <= 0 && mrp > 0 {
			salePrice = mrp
		}
		if mrp <= 0 {
			mrp = 50.0
			salePrice = 45.0
		}

		unit := extractUnit(name)
		hsn, gst, imgUrl := getCategoryAttrs(cat, subCat)

		// Generate EAN-13 barcode
		barcode := generateValidEAN13(idx)
		idx++

		seenNames[nameLower] = true
		seenBarcodes[barcode] = true

		categoryLabel := subCat
		if categoryLabel == "" {
			categoryLabel = cat
		}

		row := []string{
			barcode,
			name,
			brand,
			categoryLabel,
			unit,
			fmt.Sprintf("%.2f", mrp),
			fmt.Sprintf("%.2f", salePrice),
			imgUrl,
			hsn,
			fmt.Sprintf("%.1f", gst),
			desc,
		}

		allRows = append(allRows, row)
		addedCount++
	}

	// 3. Write final unified thousands product CSV
	outF, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("❌ Cannot create output CSV: %v\n", err)
		return
	}
	defer outF.Close()

	w := csv.NewWriter(outF)
	defer w.Flush()

	_ = w.Write(header)
	for _, row := range allRows {
		_ = w.Write(row)
	}

	fmt.Printf("\n🎉 SUCCESS! Generated mega catalog with %d TOTAL PRODUCTS!\n", len(allRows))
	fmt.Printf("📁 File Saved At: %s\n", outPath)
}
