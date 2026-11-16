package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ShopLiveStatus string

const (
	StatusOpen        ShopLiveStatus = "OPEN"
	StatusClosingSoon ShopLiveStatus = "CLOSING_SOON"
	StatusClosed      ShopLiveStatus = "CLOSED"
	StatusWeeklyOff   ShopLiveStatus = "WEEKLY_OFF"
)

// LiveTimingInfo encapsulates real-time store availability, customer-facing banners, and countdowns.
type LiveTimingInfo struct {
	Status         ShopLiveStatus `json:"status"`           // OPEN, CLOSING_SOON, CLOSED, WEEKLY_OFF
	IsOpenNow      bool           `json:"is_open_now"`       // True if customer can shop right now
	StatusMessage  string         `json:"status_message"`   // Customer banner: "Open now • Closes at 9:00 PM"
	NextChangeAt   string         `json:"next_change_at"`   // "Closes in 22 mins" or "Opens tomorrow at 9:00 AM"
	MinutesToClose int            `json:"minutes_to_close"` // Remaining open minutes (when OPEN or CLOSING_SOON)
}

// Format12HourTime converts "HH:MM" (24-hour) string into friendly 12-hour AM/PM format (e.g. "09:00" -> "9:00 AM", "21:30" -> "9:30 PM").
func Format12HourTime(timeStr string) string {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return ""
	}
	up := strings.ToUpper(timeStr)
	if strings.HasSuffix(up, "AM") || strings.HasSuffix(up, "PM") {
		return timeStr
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return timeStr
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil {
		return timeStr
	}

	period := "AM"
	displayH := h
	if h >= 12 {
		period = "PM"
		if h > 12 {
			displayH = h - 12
		}
	} else if h == 0 {
		displayH = 12
	}

	if m == 0 {
		return fmt.Sprintf("%d:00 %s", displayH, period)
	}
	return fmt.Sprintf("%d:%02d %s", displayH, m, period)
}

// parseMinutesFromMidnight converts "HH:MM" string to minutes from 00:00 (0 to 1439), robust to AM/PM and spacing.
func parseMinutesFromMidnight(timeStr string, defaultMinutes int) int {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return defaultMinutes
	}

	up := strings.ToUpper(timeStr)
	isPM := strings.Contains(up, "PM")
	isAM := strings.Contains(up, "AM")
	cleanStr := strings.ReplaceAll(timeStr, "AM", "")
	cleanStr = strings.ReplaceAll(cleanStr, "PM", "")
	cleanStr = strings.ReplaceAll(cleanStr, "am", "")
	cleanStr = strings.ReplaceAll(cleanStr, "pm", "")
	cleanStr = strings.TrimSpace(cleanStr)

	parts := strings.Split(cleanStr, ":")
	if len(parts) < 2 {
		return defaultMinutes
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return defaultMinutes
	}

	if isPM && h < 12 {
		h += 12
	} else if isAM && h == 12 {
		h = 0
	}

	return h*60 + m
}

// IsDayWeeklyOff checks whether target weekday matches shop's configured weekly off (supports English & Hindi names).
func IsDayWeeklyOff(targetWeekday time.Weekday, weeklyOffStr string) bool {
	raw := strings.ToLower(strings.TrimSpace(weeklyOffStr))
	if raw == "" || raw == "none" || raw == "never" {
		return false
	}

	// Normalize weekday names in English and common Hindi transliterations
	var targetNames []string
	switch targetWeekday {
	case time.Sunday:
		targetNames = []string{"sunday", "sun", "ravivar", "itwar", "itvaar"}
	case time.Monday:
		targetNames = []string{"monday", "mon", "somvar", "somvaar"}
	case time.Tuesday:
		targetNames = []string{"tuesday", "tue", "mangalvar", "mangalvaar"}
	case time.Wednesday:
		targetNames = []string{"wednesday", "wed", "budhvar", "budhvaar"}
	case time.Thursday:
		targetNames = []string{"thursday", "thu", "guruvar", "guruvaar", "brihaspativar"}
	case time.Friday:
		targetNames = []string{"friday", "fri", "shukravar", "shukravaar", "jumma"}
	case time.Saturday:
		targetNames = []string{"saturday", "sat", "shanivar", "shanivaar"}
	}

	for _, name := range targetNames {
		if strings.Contains(raw, name) {
			return true
		}
	}
	return false
}

// CalculateStoreLiveStatus evaluates the real-time open/close state of a shop against current Indian Standard Time (IST).
func CalculateStoreLiveStatus(isOpenManual bool, openTimeStr, closeTimeStr, weeklyOffStr string) LiveTimingInfo {
	return CalculateStoreLiveStatusAt(isOpenManual, openTimeStr, closeTimeStr, weeklyOffStr, IndianNow())
}

// CalculateStoreLiveStatusAt evaluates open/close status against a given target time (critical for deterministic unit testing).
func CalculateStoreLiveStatusAt(isOpenManual bool, openTimeStr, closeTimeStr, weeklyOffStr string, targetTime time.Time) LiveTimingInfo {
	openFormatted := Format12HourTime(openTimeStr)
	closeFormatted := Format12HourTime(closeTimeStr)
	if openFormatted == "" {
		openFormatted = "9:00 AM"
	}
	if closeFormatted == "" {
		closeFormatted = "9:00 PM"
	}

	// 1. Weekly holiday check
	if IsDayWeeklyOff(targetTime.Weekday(), weeklyOffStr) {
		return LiveTimingInfo{
			Status:         StatusWeeklyOff,
			IsOpenNow:      false,
			StatusMessage:  "Closed • Weekly holiday today",
			NextChangeAt:   fmt.Sprintf("Opens tomorrow at %s", openFormatted),
			MinutesToClose: 0,
		}
	}

	// 2. Manual Emergency Switch (owner manually closed store via app switch)
	if !isOpenManual {
		return LiveTimingInfo{
			Status:         StatusClosed,
			IsOpenNow:      false,
			StatusMessage:  "Temporarily closed by store owner",
			NextChangeAt:   fmt.Sprintf("Opens tomorrow at %s", openFormatted),
			MinutesToClose: 0,
		}
	}

	// 3. Operating hours parsing (default 09:00 to 21:00)
	openMin := parseMinutesFromMidnight(openTimeStr, 9*60)
	closeMin := parseMinutesFromMidnight(closeTimeStr, 21*60)
	currentMin := targetTime.Hour()*60 + targetTime.Minute()

	// Special Case: 24-Hour store (00:00 to 00:00 or 00:00 to 23:59)
	if (openMin == 0 && closeMin == 0) || (openMin == 0 && closeMin >= 1439) {
		return LiveTimingInfo{
			Status:         StatusOpen,
			IsOpenNow:      true,
			StatusMessage:  "Open 24 hours",
			NextChangeAt:   "Open 24 hours",
			MinutesToClose: 1440,
		}
	}

	isInside := false
	minutesToClose := 0

	if openMin < closeMin {
		// Normal daytime shift (e.g. 09:00 to 21:00)
		if currentMin >= openMin && currentMin < closeMin {
			isInside = true
			minutesToClose = closeMin - currentMin
		}
	} else {
		// Overnight shift (e.g. 21:00 to 04:00)
		if currentMin >= openMin || currentMin < closeMin {
			isInside = true
			if currentMin >= openMin {
				minutesToClose = (1440 - currentMin) + closeMin
			} else {
				minutesToClose = closeMin - currentMin
			}
		}
	}

	if isInside {
		if minutesToClose <= 30 && minutesToClose > 0 {
			return LiveTimingInfo{
				Status:         StatusClosingSoon,
				IsOpenNow:      true,
				StatusMessage:  fmt.Sprintf("Closing soon • %d mins left", minutesToClose),
				NextChangeAt:   fmt.Sprintf("Closes at %s", closeFormatted),
				MinutesToClose: minutesToClose,
			}
		}
		return LiveTimingInfo{
			Status:         StatusOpen,
			IsOpenNow:      true,
			StatusMessage:  fmt.Sprintf("Open now • Closes at %s", closeFormatted),
			NextChangeAt:   fmt.Sprintf("Closes at %s", closeFormatted),
			MinutesToClose: minutesToClose,
		}
	}

	// Outside operating hours
	nextOpenMsg := fmt.Sprintf("Opens today at %s", openFormatted)
	if openMin < closeMin {
		if currentMin >= closeMin {
			nextOpenMsg = fmt.Sprintf("Opens tomorrow at %s", openFormatted)
		}
	} else {
		// In overnight shift, if currentMin >= closeMin && currentMin < openMin
		nextOpenMsg = fmt.Sprintf("Opens today at %s", openFormatted)
	}

	return LiveTimingInfo{
		Status:         StatusClosed,
		IsOpenNow:      false,
		StatusMessage:  fmt.Sprintf("Closed • %s", nextOpenMsg),
		NextChangeAt:   nextOpenMsg,
		MinutesToClose: 0,
	}
}
