package humanize

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FormatComma formats an integer with standard Western commas every three digits (e.g. 1,234,567).
func FormatComma(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	str := strconv.FormatInt(n, 10)
	length := len(str)
	if length <= 3 {
		return sign + str
	}

	var sb strings.Builder
	sb.WriteString(sign)

	head := length % 3
	if head > 0 {
		sb.WriteString(str[:head])
		sb.WriteByte(',')
	}

	for i := head; i < length; i += 3 {
		sb.WriteString(str[i : i+3])
		if i+3 < length {
			sb.WriteByte(',')
		}
	}

	return sb.String()
}

// FormatIndian formats an integer according to the Indian numbering system
// where the first group has 3 digits and subsequent groups have 2 digits (e.g. 12,34,56,789).
func FormatIndian(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	str := strconv.FormatInt(n, 10)
	length := len(str)
	if length <= 3 {
		return sign + str
	}

	// Last 3 digits
	lastThree := str[length-3:]
	remaining := str[:length-3]

	var sb strings.Builder
	sb.WriteString(sign)

	remLen := len(remaining)
	head := remLen % 2
	if head > 0 {
		sb.WriteString(remaining[:head])
		sb.WriteByte(',')
	}

	for i := head; i < remLen; i += 2 {
		sb.WriteString(remaining[i : i+2])
		sb.WriteByte(',')
	}

	sb.WriteString(lastThree)
	return sb.String()
}

// Compact formats large numbers into short human strings (e.g. 1.5K, 2.4M, 1.2B).
func Compact(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	fn := float64(n)

	switch {
	case fn >= 1e12:
		return sign + formatCompactFloat(fn/1e12) + "T"
	case fn >= 1e9:
		return sign + formatCompactFloat(fn/1e9) + "B"
	case fn >= 1e6:
		return sign + formatCompactFloat(fn/1e6) + "M"
	case fn >= 1e3:
		return sign + formatCompactFloat(fn/1e3) + "K"
	default:
		return sign + strconv.FormatInt(n, 10)
	}
}

// CompactIndian formats numbers using Indian terms (Crore, Lakh, K).
func CompactIndian(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	fn := float64(n)

	switch {
	case fn >= 1e7: // 1 Crore
		return sign + formatCompactFloat(fn/1e7) + " Cr"
	case fn >= 1e5: // 1 Lakh
		return sign + formatCompactFloat(fn/1e5) + " Lakh"
	case fn >= 1e3: // 1 Thousand
		return sign + formatCompactFloat(fn/1e3) + " K"
	default:
		return sign + strconv.FormatInt(n, 10)
	}
}

func formatCompactFloat(f float64) string {
	rounded := math.Round(f*10) / 10
	if rounded == math.Floor(rounded) {
		return strconv.FormatInt(int64(rounded), 10)
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// Ordinal formats an integer with its English ordinal suffix (e.g. 1st, 2nd, 3rd, 4th, 21st).
func Ordinal(n int) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}

	// 11th, 12th, 13th exception
	rem100 := abs % 100
	if rem100 >= 11 && rem100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}

	switch abs % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// TimeAgo converts a past timestamp into a relative human duration string (e.g. "just now", "5 minutes ago").
func TimeAgo(t time.Time) string {
	diff := time.Since(t)
	if diff < 0 {
		return "in the future"
	}

	seconds := int(diff.Seconds())
	minutes := int(diff.Minutes())
	hours := int(diff.Hours())
	days := hours / 24

	switch {
	case seconds < 15:
		return "just now"
	case seconds < 60:
		return fmt.Sprintf("%d seconds ago", seconds)
	case minutes == 1:
		return "1 minute ago"
	case minutes < 60:
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours == 1:
		return "1 hour ago"
	case hours < 24:
		return fmt.Sprintf("%d hours ago", hours)
	case days == 1:
		return "yesterday"
	case days < 30:
		return fmt.Sprintf("%d days ago", days)
	case days < 60:
		return "1 month ago"
	case days < 365:
		return fmt.Sprintf("%d months ago", days/30)
	case days < 730:
		return "1 year ago"
	default:
		return fmt.Sprintf("%d years ago", days/365)
	}
}

// Plural formats a count with its singular or plural label (e.g. "1 item" vs "4 items").
func Plural(count int, singular, plural string) string {
	if count == 1 || count == -1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
