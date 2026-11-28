package utils

import (
	"context"
	"testing"
	"time"
)

func TestResolveLocation(t *testing.T) {
	// 1. Valid IANA name
	loc := ResolveLocation("Asia/Kolkata", "")
	if loc == nil {
		t.Fatal("expected non-nil location for Asia/Kolkata")
	}
	_, off := time.Now().In(loc).Zone()
	if off != 5*3600+1800 {
		t.Fatalf("expected +19800s offset for Asia/Kolkata, got %d", off)
	}

	// 2. Numeric offset fallback (e.g. -300 = UTC-05:00)
	est := ResolveLocation("INVALID/ZONE", "-300")
	_, estOff := time.Now().In(est).Zone()
	if estOff != -300*60 {
		t.Fatalf("expected -18000s offset, got %d", estOff)
	}

	// 3. Empty/invalid falls back to IndianLocation
	def := ResolveLocation("", "not-a-number")
	if def != IndianLocation {
		t.Fatalf("expected fallback to IndianLocation, got %v", def)
	}
}

func TestContextTimezoneHelpers(t *testing.T) {
	tokyo := ResolveLocation("", "540") // UTC+09:00
	ctx := WithLocation(context.Background(), tokyo)

	utcTime := time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC)
	converted := InUserZone(ctx, utcTime)
	if converted.Hour() != 9 {
		t.Fatalf("expected 09:00 in UTC+9, got %02d:00", converted.Hour())
	}

	nowUser := NowInContext(ctx)
	_, off := nowUser.Zone()
	if off != 540*60 {
		t.Fatalf("expected +32400s offset from NowInContext, got %d", off)
	}
}
