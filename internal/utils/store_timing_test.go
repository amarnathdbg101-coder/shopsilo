package utils_test

import (
	"testing"
	"time"

	"shopMe/internal/utils"
)

func TestFormat12HourTime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"09:00", "9:00 AM"},
		{"09:30", "9:30 AM"},
		{"12:00", "12:00 PM"},
		{"12:45", "12:45 PM"},
		{"13:00", "1:00 PM"},
		{"21:00", "9:00 PM"},
		{"21:15", "9:15 PM"},
		{"00:00", "12:00 AM"},
		{"00:30", "12:30 AM"},
		{"9:00 AM", "9:00 AM"},
		{"10:30 PM", "10:30 PM"},
		{"9:00", "9:00 AM"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := utils.Format12HourTime(tt.input)
			if got != tt.expected {
				t.Errorf("Format12HourTime(%s) = %s, expected %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsDayWeeklyOff(t *testing.T) {
	tests := []struct {
		day      time.Weekday
		input    string
		expected bool
	}{
		{time.Sunday, "Sunday", true},
		{time.Sunday, "sunday", true},
		{time.Sunday, "Ravivar", true},
		{time.Sunday, "itwar", true},
		{time.Sunday, "Wednesday", false},
		{time.Wednesday, "Wednesday, Sunday", true},
		{time.Monday, "Somvar", true},
		{time.Friday, "Jumma", true},
		{time.Tuesday, "", false},
		{time.Saturday, "none", false},
	}

	for _, tt := range tests {
		got := utils.IsDayWeeklyOff(tt.day, tt.input)
		if got != tt.expected {
			t.Errorf("IsDayWeeklyOff(%v, %s) = %v, expected %v", tt.day, tt.input, got, tt.expected)
		}
	}
}

func TestCalculateStoreLiveStatus(t *testing.T) {
	loc := utils.IndianLocation

	t.Run("Normal Store Open During Daytime (09:00 to 21:00 at 14:00 Wednesday)", func(t *testing.T) {
		// Wednesday, 14:00 IST
		target := time.Date(2026, 11, 4, 14, 0, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusOpen {
			t.Errorf("expected StatusOpen, got %v", info.Status)
		}
		if !info.IsOpenNow {
			t.Errorf("expected IsOpenNow = true")
		}
		if info.MinutesToClose != 420 { // 7 hours = 420 mins
			t.Errorf("expected 420 mins to close, got %d", info.MinutesToClose)
		}
	})

	t.Run("Normal Store Closing Soon (at 20:45, closes at 21:00)", func(t *testing.T) {
		// Wednesday, 20:45 IST
		target := time.Date(2026, 11, 4, 20, 45, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusClosingSoon {
			t.Errorf("expected StatusClosingSoon, got %v", info.Status)
		}
		if !info.IsOpenNow {
			t.Errorf("expected IsOpenNow = true")
		}
		if info.MinutesToClose != 15 {
			t.Errorf("expected 15 mins to close, got %d", info.MinutesToClose)
		}
		if info.StatusMessage != "Closing soon • 15 mins left" {
			t.Errorf("unexpected StatusMessage: %s", info.StatusMessage)
		}
	})

	t.Run("Normal Store Closed at Night (at 22:30, closes at 21:00)", func(t *testing.T) {
		// Wednesday, 22:30 IST
		target := time.Date(2026, 11, 4, 22, 30, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusClosed {
			t.Errorf("expected StatusClosed, got %v", info.Status)
		}
		if info.IsOpenNow {
			t.Errorf("expected IsOpenNow = false")
		}
		if info.NextChangeAt != "Opens tomorrow at 9:00 AM" {
			t.Errorf("expected 'Opens tomorrow at 9:00 AM', got '%s'", info.NextChangeAt)
		}
	})

	t.Run("Normal Store Closed in Early Morning (at 07:30, opens at 09:00)", func(t *testing.T) {
		// Wednesday, 07:30 IST
		target := time.Date(2026, 11, 4, 7, 30, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusClosed {
			t.Errorf("expected StatusClosed, got %v", info.Status)
		}
		if info.IsOpenNow {
			t.Errorf("expected IsOpenNow = false")
		}
		if info.NextChangeAt != "Opens today at 9:00 AM" {
			t.Errorf("expected 'Opens today at 9:00 AM', got '%s'", info.NextChangeAt)
		}
	})

	t.Run("Weekly Off Day (Sunday Holiday)", func(t *testing.T) {
		// Sunday, 14:00 IST
		target := time.Date(2026, 11, 8, 14, 0, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusWeeklyOff {
			t.Errorf("expected StatusWeeklyOff, got %v", info.Status)
		}
		if info.IsOpenNow {
			t.Errorf("expected IsOpenNow = false")
		}
		if info.StatusMessage != "Closed • Weekly holiday today" {
			t.Errorf("unexpected StatusMessage: %s", info.StatusMessage)
		}
	})

	t.Run("Emergency Switch Closed by Owner (is_open = false)", func(t *testing.T) {
		target := time.Date(2026, 11, 4, 14, 0, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(false, "09:00", "21:00", "Sunday", target)

		if info.Status != utils.StatusClosed {
			t.Errorf("expected StatusClosed, got %v", info.Status)
		}
		if info.IsOpenNow {
			t.Errorf("expected IsOpenNow = false")
		}
		if info.StatusMessage != "Temporarily closed by store owner" {
			t.Errorf("unexpected StatusMessage: %s", info.StatusMessage)
		}
	})

	t.Run("Overnight Pharmacy (21:00 to 04:00 at 02:30 IST)", func(t *testing.T) {
		// 02:30 AM IST inside overnight shift
		target := time.Date(2026, 11, 5, 2, 30, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "21:00", "04:00", "", target)

		if info.Status != utils.StatusOpen {
			t.Errorf("expected StatusOpen, got %v", info.Status)
		}
		if !info.IsOpenNow {
			t.Errorf("expected IsOpenNow = true")
		}
		if info.MinutesToClose != 90 { // 02:30 to 04:00 = 90 mins
			t.Errorf("expected 90 mins to close, got %d", info.MinutesToClose)
		}
	})

	t.Run("Overnight Pharmacy Closing Soon (at 03:40, closes at 04:00)", func(t *testing.T) {
		target := time.Date(2026, 11, 5, 3, 40, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "21:00", "04:00", "", target)

		if info.Status != utils.StatusClosingSoon {
			t.Errorf("expected StatusClosingSoon, got %v", info.Status)
		}
		if info.MinutesToClose != 20 {
			t.Errorf("expected 20 mins to close, got %d", info.MinutesToClose)
		}
	})

	t.Run("24 Hours Store (00:00 to 00:00)", func(t *testing.T) {
		target := time.Date(2026, 11, 4, 3, 15, 0, 0, loc)
		info := utils.CalculateStoreLiveStatusAt(true, "00:00", "00:00", "", target)

		if info.Status != utils.StatusOpen {
			t.Errorf("expected StatusOpen, got %v", info.Status)
		}
		if !info.IsOpenNow {
			t.Errorf("expected IsOpenNow = true")
		}
		if info.StatusMessage != "Open 24 hours" {
			t.Errorf("expected 'Open 24 hours', got '%s'", info.StatusMessage)
		}
	})
}
