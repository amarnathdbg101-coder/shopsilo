package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// OpenFoodFactsProduct defines the JSON schema from Open Food Facts API v2.
type OpenFoodFactsProduct struct {
	Code            string `json:"code"`
	ProductName     string `json:"product_name"`
	Brands          string `json:"brands"`
	Categories      string `json:"categories"`
	Quantity        string `json:"quantity"`
	ImageFrontURL   string `json:"image_front_url"`
	ImageURL        string `json:"image_url"`
}

type OpenFoodFactsResponse struct {
	Count    int                    `json:"count"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Products []OpenFoodFactsProduct `json:"products"`
}

func main() {
	pages := flag.Int("pages", 5, "Number of pages to fetch (100 products per page)")
	country := flag.String("country", "india", "Target country tag (e.g. india, united-kingdom, world)")
	outFile := flag.String("out", "data/open_catalog_downloaded.csv", "Output CSV file destination")
	flag.Parse()

	fmt.Printf("🌐 Fetching real product barcodes from Open Food Facts for country='%s' (%d pages)...\n", *country, *pages)

	f, err := os.Create(*outFile)
	if err != nil {
		fmt.Printf("❌ Failed to create CSV file: %v\n", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// ShopSilo standard CSV header
	_ = writer.Write([]string{
		"barcode", "name", "brand", "category", "unit",
		"mrp", "selling_price", "image_url", "hsn", "gst", "description",
	})

	client := &http.Client{Timeout: 15 * time.Second}
	totalSaved := 0

	for page := 1; page <= *pages; page++ {
		endpoint := fmt.Sprintf(
			"https://world.openfoodfacts.org/cgi/search.pl?action=process&tagtype_0=countries&tag_contains_0=contains&tag_0=%s&json=1&page_size=100&page=%d&fields=code,product_name,brands,categories,quantity,image_front_url,image_url",
			url.QueryEscape(*country),
			page,
		)

		var bodyBytes []byte
		var success bool
		for attempt := 1; attempt <= 3; attempt++ {
			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				break
			}
			req.Header.Set("User-Agent", "ShopSilo-Enterprise-MasterCatalog/2.0 (contact: support@shopsilo.in)")

			resp, err := client.Do(req)
			if err != nil {
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			if resp.StatusCode == http.StatusOK {
				bodyBytes, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
				success = true
				break
			}
			resp.Body.Close()
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
		if !success {
			fmt.Printf("⚠️ Skipped page %d after 3 retries (rate limit)\n", page)
			continue
		}

		var offResp OpenFoodFactsResponse
		if err := json.Unmarshal(bodyBytes, &offResp); err != nil {
			fmt.Printf("⚠️ JSON unmarshal error on page %d: %v\n", page, err)
			continue
		}

		if len(offResp.Products) == 0 {
			fmt.Println("ℹ️ No more products returned by API.")
			break
		}

		for _, p := range offResp.Products {
			code := strings.TrimSpace(p.Code)
			name := strings.TrimSpace(p.ProductName)
			if code == "" || name == "" {
				continue
			}

			// Clean brand & category
			brand := strings.TrimSpace(p.Brands)
			if strings.Contains(brand, ",") {
				brand = strings.TrimSpace(strings.Split(brand, ",")[0])
			}

			cat := "Packaged Foods"
			if p.Categories != "" {
				parts := strings.Split(p.Categories, ",")
				if len(parts) > 0 {
					cat = strings.TrimSpace(parts[0])
				}
			}

			unit := strings.TrimSpace(p.Quantity)
			if unit == "" {
				unit = "piece"
			}

			img := strings.TrimSpace(p.ImageFrontURL)
			if img == "" {
				img = strings.TrimSpace(p.ImageURL)
			}

			// Default GST and HSN estimation based on category
			hsn := "19053100"
			gst := 18.0
			catLow := strings.ToLower(cat)
			if strings.Contains(catLow, "oil") || strings.Contains(catLow, "ghee") {
				hsn = "15121910"
				gst = 5.0
			} else if strings.Contains(catLow, "tea") || strings.Contains(catLow, "coffee") {
				hsn = "09024020"
				gst = 5.0
			} else if strings.Contains(catLow, "noodle") || strings.Contains(catLow, "pasta") {
				hsn = "19023010"
				gst = 12.0
			} else if strings.Contains(catLow, "dairy") || strings.Contains(catLow, "butter") {
				hsn = "04051000"
				gst = 12.0
			}

			record := []string{
				code,
				name,
				brand,
				cat,
				unit,
				"0.00", // MRP placeholder (shop owners set their own)
				"0.00", // Selling Price placeholder
				img,
				hsn,
				strconv.FormatFloat(gst, 'f', 1, 64),
				fmt.Sprintf("Authentic FMCG barcode from Open Food Facts catalog (%s)", brand),
			}

			_ = writer.Write(record)
			totalSaved++
		}

		fmt.Printf("✅ Page %d complete (%d total products written)...\n", page, totalSaved)
		time.Sleep(500 * time.Millisecond) // Polite delay between requests
	}

	fmt.Printf("\n🎉 Finished! Saved %d live products to %s\n", totalSaved, *outFile)
	fmt.Println("🚀 You can now seed this file directly into PostgreSQL using:")
	fmt.Printf("   go run ./cmd/seed_catalog -csv=\"%s\"\n", *outFile)
}
