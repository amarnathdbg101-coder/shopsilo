package utils

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"shopMe/internal/handler/model"
)

func TestResolveLocation_2000PredictableAndUnpredictableInputs(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	weirdZones := []string{
		"", "   ", "Asia/Kolkata", "UTC", "America/New_York", "Europe/London",
		"INVALID/ZONE", "IST", "DROP TABLE pos_bills;", "\x00\xff", "🚀",
		"Mars/Olympus_Mons", "Asia/Tokyo", "Pacific/Auckland",
	}
	weirdOffsets := []string{
		"", "330", "-300", "0", "840", "-720", "99999", "-99999", "abc", "12.5", "NaN",
	}

	for i := 0; i < 2000; i++ {
		tz := weirdZones[rng.Intn(len(weirdZones))]
		off := weirdOffsets[rng.Intn(len(weirdOffsets))]
		if i%5 == 0 {
			off = fmt.Sprintf("%d", rng.Intn(3000)-1500)
		}
		loc := ResolveLocation(tz, off)
		if loc == nil {
			t.Fatalf("iteration %d: ResolveLocation(%q, %q) returned nil", i, tz, off)
		}
		ctx := WithLocation(context.Background(), loc)
		_ = NowInContext(ctx).Format(time.RFC3339)
		_ = InUserZone(ctx, time.Time{}).Format("02-Jan-2006")
	}
}

func TestNativePDF_StressAndEdgeCases(t *testing.T) {
	// Concurrent 50-goroutine stress test with unpredictable bill shapes
	var wg sync.WaitGroup
	rng := rand.New(rand.NewSource(99))

	for g := 0; g < 50; g++ {
		g := g
		seed := rng.Int63()
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			itemCount := r.Intn(25)
			items := make([]*model.POSBillItem, 0, itemCount)
			for j := 0; j < itemCount; j++ {
				items = append(items, &model.POSBillItem{
					ProductName: fmt.Sprintf("दाल / Item #%d-%d ₹", g, j),
					Quantity:    r.Intn(20) + 1,
					UnitPrice:   float64(r.Intn(5000)) + 0.5,
					TotalPrice:  float64(r.Intn(10000)) + 1.0,
				})
			}
			bill := &model.POSBill{
				BillNumber:     fmt.Sprintf("BIL-STRESS-%d", g),
				Shop:           &model.Shop{Name: "गुप्ता किराना स्टोर • Stress Shop"},
				CustomerPhone:  "9876543210",
				Subtotal:       1250.50,
				DiscountAmount: float64(r.Intn(100)),
				TotalAmount:    1200.50,
				PaymentMethod:  "split",
				CashAmount:     700,
				OnlineAmount:   500.50,
				Items:          items,
			}
			if g%3 == 0 {
				bill.CreatedAt = time.Time{} // zero time edge case
			} else {
				bill.CreatedAt = time.Now().UTC().Add(-time.Duration(r.Intn(500)) * time.Hour)
			}

			pdfBytes, err := GenerateNativeBillPDF(bill, NativePDFOptions{Format: FormatA5})
			if err != nil || len(pdfBytes) < 500 {
				t.Errorf("A5 PDF failed for g=%d: err=%v len=%d", g, err, len(pdfBytes))
			}
			thermalBytes, err := GenerateNativeThermalBillPDF(bill)
			if err != nil || len(thermalBytes) < 300 {
				t.Errorf("Thermal PDF failed for g=%d: err=%v len=%d", g, err, len(thermalBytes))
			}
		}()
	}
	wg.Wait()
}
