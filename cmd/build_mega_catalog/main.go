package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Mega catalog builder: Merges verified core products, Indian FMCG brands, and live Open Food Facts feeds.
func main() {
	outFile := "data/master_catalog_mega.csv"
	fmt.Printf("🔨 Building consolidated Master Catalog Mega Dataset -> %s\n", outFile)

	seenBarcodes := make(map[string]bool)
	var allRows [][]string

	header := []string{
		"barcode", "name", "brand", "category", "unit",
		"mrp", "selling_price", "image_url", "hsn", "gst", "description",
	}

	// 1. Process local CSV files in data/
	inputFiles := []string{
		"data/master_catalog_fmcg.csv",
		"data/open_catalog_downloaded.csv",
		"data/test_fetch.csv",
	}

	for _, path := range inputFiles {
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		r := csv.NewReader(f)
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = true

		// Read header
		_, _ = r.Read()

		count := 0
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil || len(rec) < 2 {
				continue
			}

			barcode := strings.TrimSpace(rec[0])
			name := strings.TrimSpace(rec[1])
			if barcode == "" || name == "" {
				continue
			}

			cleanBC := strings.ToLower(barcode)
			if seenBarcodes[cleanBC] {
				continue
			}
			seenBarcodes[cleanBC] = true

			// Ensure at least 11 columns
			row := make([]string, 11)
			copy(row, rec)
			if row[4] == "" {
				row[4] = "piece"
			}
			if row[8] == "" {
				row[8] = "19053100"
			}
			if row[9] == "" {
				row[9] = "18.0"
			}

			allRows = append(allRows, row)
			count++
		}
		f.Close()
		fmt.Printf("✅ Loaded %d unique products from %s\n", count, path)
	}

	// 2. Write combined deduplicated mega catalog
	outF, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("❌ Failed to create output file: %v\n", err)
		return
	}
	defer outF.Close()

	w := csv.NewWriter(outF)
	defer w.Flush()

	_ = w.Write(header)
	for _, row := range allRows {
		_ = w.Write(row)
	}

	fmt.Printf("\n🎉 Mega Catalog Successfully Created! Total Unique Products: %d\n", len(allRows))
	fmt.Printf("📁 Destination: %s\n", outFile)
	fmt.Println("🚀 Now seeding into PostgreSQL master_catalog_products...")
}
