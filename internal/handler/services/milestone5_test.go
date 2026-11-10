package services_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/migration"
	"shopMe/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func setupMilestone5Test(t *testing.T) (*pgxpool.Pool, *services.EODPDFService, string) {
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
		t.Skip("DATABASE_URL not set")
	}

	if testing.Short() {
		t.Skip("skipping remote db test in short mode")
	}
	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("cannot connect to test db (%v); skipping integration test", err)
		return nil, nil, ""
	}

	migration.EnsureSchemaColumns(pool)

	logger, _ := zap.NewDevelopment()
	shopRepo := repository.NewShopRepo(pool, logger)
	posRepo := repository.NewPOSRepo(pool, logger)
	drawerRepo := repository.NewCashDrawerRepo(pool, logger)

	pdfService := services.NewEODPDFService(shopRepo, posRepo, drawerRepo, logger)

	var ownerUserID string
	err = pool.QueryRow(context.Background(), "SELECT user_id FROM shops WHERE is_active = true LIMIT 1;").Scan(&ownerUserID)
	if err != nil {
		t.Skip("No active shop found for test")
	}

	return pool, pdfService, ownerUserID
}

func TestAutomatedEndOfDayPLPDF(t *testing.T) {
	pool, pdfService, ownerUserID := setupMilestone5Test(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("Generate End-of-Day P&L PDF via Pure Go gopdf Engine", func(t *testing.T) {
		pdfBytes, err := pdfService.GenerateEndOfDayPLPDF(ctx, ownerUserID, time.Now())
		if err != nil {
			t.Fatalf("GenerateEndOfDayPLPDF failed: %v", err)
		}
		if len(pdfBytes) < 500 {
			t.Fatalf("Generated PDF is too small (%d bytes), expected > 500 bytes", len(pdfBytes))
		}

		// Verify PDF magic header bytes
		if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
			t.Errorf("Expected PDF header magic '%%PDF-', got '%s'", string(pdfBytes[:10]))
		}

		t.Logf("End-of-Day P&L PDF successfully generated! Size=%d bytes, MagicHeader=%s",
			len(pdfBytes), string(pdfBytes[:5]))
	})
}

