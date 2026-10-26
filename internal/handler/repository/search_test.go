package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestTypoAndFuzzyProductSearch(t *testing.T) {
	_ = os.Setenv("PORT", "8080")
	// Try loading .env from parent directories
	for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					_ = os.Setenv("DATABASE_URL", strings.TrimPrefix(line, "DATABASE_URL="))
				}
			}
			break
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping live database test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	repo := repository.NewProductRepo(pool, logger)

	testCases := []struct {
		name       string
		searchTerm string
		expectMin  int
	}{
		{
			name:       "Exact search for shampoo",
			searchTerm: "shampoo",
			expectMin:  1,
		},
		{
			name:       "Typo search for clinic (clnic)",
			searchTerm: "clnic",
			expectMin:  1,
		},
		{
			name:       "Typo search for pantene (pantn)",
			searchTerm: "pantn",
			expectMin:  1,
		},
		{
			name:       "Multi-word typo search (clnic shampu)",
			searchTerm: "clnic shampu",
			expectMin:  1,
		},
		{
			name:       "Bilingual Hinglish search for hair oil (tel)",
			searchTerm: "tel",
			expectMin:  1,
		},
		{
			name:       "Non-existent nonsense query",
			searchTerm: "xyznotfound999",
			expectMin:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := dto.ProductFilter{
				Search: tc.searchTerm,
				Limit:  10,
				Page:   1,
			}
			products, totalCount, err := repo.FindAll(ctx, filter)
			if err != nil {
				t.Fatalf("FindAll failed for query '%s': %v", tc.searchTerm, err)
			}
			t.Logf("Query: '%s' -> Found %d products (total: %d)", tc.searchTerm, len(products), totalCount)
			for i, p := range products {
				if i < 3 {
					t.Logf("  [%d] %s (₹%.2f)", i+1, p.Name, p.Price)
				}
			}
			if totalCount < tc.expectMin {
				t.Errorf("Expected at least %d product for query '%s', got %d", tc.expectMin, tc.searchTerm, totalCount)
			}
		})
	}
}
