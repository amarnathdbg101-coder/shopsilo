package bizhours

import (
	"testing"
	"time"
)

func TestBizHours_DaytimeShift(t *testing.T) {
	// Thursday, 14:00 (2:00 PM)
	now := time.Date(2026, 11, 5, 14, 0, 0, 0, time.UTC)

	res := CalculateAt(true, "09:00", "21:00", "Sunday", now)
	if !res.IsOpenNow || res.Status != StatusOpen {
		t.Fatalf("expected store to be OPEN, got %s", res.Status)
	}
	if res.MinutesToClose != 7*60 { // 14:00 to 21:00 = 420 mins
		t.Fatalf("expected 420 mins to close, got %d", res.MinutesToClose)
	}
}

func TestBizHours_ClosingSoon(t *testing.T) {
	// Thursday, 20:45 (8:45 PM), closes at 21:00 (15 mins left)
	now := time.Date(2026, 11, 5, 20, 45, 0, 0, time.UTC)

	res := CalculateAt(true, "09:00", "21:00", "Sunday", now)
	if !res.IsOpenNow || res.Status != StatusClosingSoon {
		t.Fatalf("expected store to be CLOSING_SOON, got %s", res.Status)
	}
	if res.MinutesToClose != 15 {
		t.Fatalf("expected 15 mins to close, got %d", res.MinutesToClose)
	}
}

func TestBizHours_OvernightShift(t *testing.T) {
	// Shift from 22:00 to 04:00
	// Test at 01:30 AM (next calendar day)
	now := time.Date(2026, 11, 6, 1, 30, 0, 0, time.UTC)

	res := CalculateAt(true, "22:00", "04:00", "None", now)
	if !res.IsOpenNow || res.Status != StatusOpen {
		t.Fatalf("expected overnight shift to be OPEN at 01:30 AM, got %s", res.Status)
	}
	if res.MinutesToClose != 150 { // 1:30 to 4:00 = 2.5 hrs = 150 mins
		t.Fatalf("expected 150 mins to close, got %d", res.MinutesToClose)
	}
}

func TestBizHours_WeeklyOff(t *testing.T) {
	// Sunday
	sunday := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	res := CalculateAt(true, "09:00", "21:00", "Sunday", sunday)
	if res.IsOpenNow || res.Status != StatusWeeklyOff {
		t.Fatalf("expected WEEKLY_OFF, got %s", res.Status)
	}
}

func TestBizHours_24Hours(t *testing.T) {
	now := time.Date(2026, 11, 5, 3, 15, 0, 0, time.UTC)

	res := CalculateAt(true, "00:00", "23:59", "None", now)
	if !res.IsOpenNow || res.Status != StatusOpen {
		t.Fatalf("expected 24-hr operation to be OPEN, got %s", res.Status)
	}
}

func TestBizHours_FormatTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"09:00", "9:00 AM"},
		{"12:00", "12:00 PM"},
		{"13:45", "1:45 PM"},
		{"21:00", "9:00 PM"},
		{"00:30", "12:30 AM"},
	}

	for _, tc := range tests {
		actual := FormatTime(tc.input)
		if actual != tc.expected {
			t.Errorf("FormatTime(%s): expected %s, got %s", tc.input, tc.expected, actual)
		}
	}
}
