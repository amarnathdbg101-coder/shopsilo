package bizhours

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseMinutesFromMidnight converts "HH:MM" (24-hr or 12-hr AM/PM) string into minutes from midnight (0..1439).
// Optimized with a zero-allocation fast-path for standard "HH:MM" formatted inputs.
func ParseMinutesFromMidnight(timeStr string, defaultMinutes int) int {
	// Zero-allocation fast-path for standard 5-character "HH:MM" (e.g. "09:00", "21:30")
	if len(timeStr) == 5 && timeStr[2] == ':' {
		h0, h1, m0, m1 := timeStr[0], timeStr[1], timeStr[3], timeStr[4]
		if h0 >= '0' && h0 <= '9' && h1 >= '0' && h1 <= '9' && m0 >= '0' && m0 <= '9' && m1 >= '0' && m1 <= '9' {
			h := int(h0-'0')*10 + int(h1-'0')
			m := int(m0-'0')*10 + int(m1-'0')
			if h <= 23 && m <= 59 {
				return h*60 + m
			}
		}
	}

	clean := strings.TrimSpace(timeStr)
	if clean == "" {
		return defaultMinutes
	}

	up := strings.ToUpper(clean)
	isPM := strings.Contains(up, "PM")
	isAM := strings.Contains(up, "AM")
	clean = strings.ReplaceAll(clean, "AM", "")
	clean = strings.ReplaceAll(clean, "PM", "")
	clean = strings.ReplaceAll(clean, "am", "")
	clean = strings.ReplaceAll(clean, "pm", "")
	clean = strings.TrimSpace(clean)

	parts := strings.Split(clean, ":")
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

// Format12Hour converts "HH:MM" string to a clean 12-hour AM/PM representation.
func Format12Hour(timeStr string) string {
	// Fast-path for standard "HH:MM"
	if len(timeStr) == 5 && timeStr[2] == ':' {
		h0, h1, m0, m1 := timeStr[0], timeStr[1], timeStr[3], timeStr[4]
		if h0 >= '0' && h0 <= '9' && h1 >= '0' && h1 <= '9' && m0 >= '0' && m0 <= '9' && m1 >= '0' && m1 <= '9' {
			h := int(h0-'0')*10 + int(h1-'0')
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
			if m0 == '0' && m1 == '0' {
				return strconv.Itoa(displayH) + ":00 " + period
			}
			return strconv.Itoa(displayH) + ":" + string([]byte{m0, m1}) + " " + period
		}
	}

	clean := strings.TrimSpace(timeStr)
	if clean == "" {
		return ""
	}
	up := strings.ToUpper(clean)
	if strings.HasSuffix(up, "AM") || strings.HasSuffix(up, "PM") {
		return clean
	}

	parts := strings.Split(clean, ":")
	if len(parts) < 2 {
		return clean
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil {
		return clean
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

// MatchesWeeklyOff tests if target weekday matches holiday string (supports English & Hindi transliterations).
func MatchesWeeklyOff(targetWeekday time.Weekday, weeklyOffStr string) bool {
	if weeklyOffStr == "" || weeklyOffStr == "none" || weeklyOffStr == "None" || weeklyOffStr == "never" {
		return false
	}

	// Fast exact matches
	switch targetWeekday {
	case time.Sunday:
		if weeklyOffStr == "Sunday" || weeklyOffStr == "sunday" || weeklyOffStr == "Sun" || weeklyOffStr == "sun" {
			return true
		}
	case time.Saturday:
		if weeklyOffStr == "Saturday" || weeklyOffStr == "saturday" || weeklyOffStr == "Sat" || weeklyOffStr == "sat" {
			return true
		}
	case time.Friday:
		if weeklyOffStr == "Friday" || weeklyOffStr == "friday" || weeklyOffStr == "Fri" || weeklyOffStr == "fri" {
			return true
		}
	case time.Monday:
		if weeklyOffStr == "Monday" || weeklyOffStr == "monday" || weeklyOffStr == "Mon" || weeklyOffStr == "mon" {
			return true
		}
	case time.Tuesday:
		if weeklyOffStr == "Tuesday" || weeklyOffStr == "tuesday" || weeklyOffStr == "Tue" || weeklyOffStr == "tue" {
			return true
		}
	case time.Wednesday:
		if weeklyOffStr == "Wednesday" || weeklyOffStr == "wednesday" || weeklyOffStr == "Wed" || weeklyOffStr == "wed" {
			return true
		}
	case time.Thursday:
		if weeklyOffStr == "Thursday" || weeklyOffStr == "thursday" || weeklyOffStr == "Thu" || weeklyOffStr == "thu" {
			return true
		}
	}

	raw := strings.ToLower(strings.TrimSpace(weeklyOffStr))
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
