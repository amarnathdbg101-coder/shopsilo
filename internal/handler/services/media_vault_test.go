package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

func TestContentHashGeneration(t *testing.T) {
	sampleImageBytes := []byte("fake-jpeg-image-binary-stream-12345")
	hashSum := sha256.Sum256(sampleImageBytes)
	hashHex := hex.EncodeToString(hashSum[:])

	if len(hashHex) != 64 {
		t.Fatalf("expected SHA-256 hash length of 64 hex characters, got %d", len(hashHex))
	}

	// Idempotent test: identical bytes must always produce identical hash
	hashSum2 := sha256.Sum256(sampleImageBytes)
	hashHex2 := hex.EncodeToString(hashSum2[:])

	if hashHex != hashHex2 {
		t.Fatalf("hashes for identical image bytes do not match: %s vs %s", hashHex, hashHex2)
	}
}

func TestProductCodeNormalization(t *testing.T) {
	rawCode := "  tat-slt-1kg  "
	normalized := strings.TrimSpace(strings.ToUpper(rawCode))

	if normalized != "TAT-SLT-1KG" {
		t.Fatalf("expected TAT-SLT-1KG, got %s", normalized)
	}
}

func TestBulkImportImageInheritanceSimulation(t *testing.T) {
	// Simulated master vault
	vaultMap := map[string]string{
		"TAT-SLT-1KG": "https://img.shopsilo.in/products/tata-salt-1kg.webp",
		"MAG-NDL-70G": "https://img.shopsilo.in/products/maggi-70g.webp",
	}

	items := []dto.BulkImportProductItem{
		{Name: "Tata Salt 1kg", SKU: "tat-slt-1kg", Price: 28},
		{Name: "Maggi Noodles", SKU: "MAG-NDL-70G", Price: 14},
		{Name: "Local Loose Sugar", SKU: "SUG-001", Price: 42},
	}

	for _, item := range items {
		cleanSKU := strings.ToUpper(strings.TrimSpace(item.SKU))
		inheritedURL, found := vaultMap[cleanSKU]

		if item.SKU == "tat-slt-1kg" {
			if !found || inheritedURL != "https://img.shopsilo.in/products/tata-salt-1kg.webp" {
				t.Fatalf("failed to inherit master image for Tata Salt, got %s", inheritedURL)
			}
		}

		if item.SKU == "SUG-001" {
			if found {
				t.Fatalf("unexpected vault match for custom SKU SUG-001")
			}
		}
	}
}

func TestProductCreationMasterImageAutoInherit(t *testing.T) {
	vaultItems := []model.ProductMediaVaultItem{
		{
			ProductCode:      "TAT-SLT-1KG",
			ImageURL:         "https://img.shopsilo.in/master/tata_salt.jpg",
			IsVerifiedMaster: true,
		},
	}

	var productImages []string
	sku := "TAT-SLT-1KG"

	if len(productImages) == 0 && sku != "" {
		for _, vi := range vaultItems {
			if vi.ImageURL != "" && len(productImages) < 4 {
				productImages = append(productImages, vi.ImageURL)
			}
		}
	}

	if len(productImages) != 1 || productImages[0] != "https://img.shopsilo.in/master/tata_salt.jpg" {
		t.Fatalf("expected product to auto-inherit vault image, got %v", productImages)
	}
}
