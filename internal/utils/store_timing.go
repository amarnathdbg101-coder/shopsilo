package utils

import (
	"time"

	"github.com/amarnathdbg101-coder/go/go-bizhours"
)

// ShopLiveStatus defines operating states for customer-facing store discovery.
type ShopLiveStatus = bizhours.Status

const (
	StatusOpen        ShopLiveStatus = bizhours.StatusOpen
	StatusClosingSoon ShopLiveStatus = bizhours.StatusClosingSoon
	StatusClosed      ShopLiveStatus = bizhours.StatusClosed
	StatusWeeklyOff   ShopLiveStatus = bizhours.StatusWeeklyOff
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
	return bizhours.Format12Hour(timeStr)
}

// IsDayWeeklyOff checks whether target weekday matches shop's configured weekly off (supports English & Hindi names).
func IsDayWeeklyOff(targetWeekday time.Weekday, weeklyOffStr string) bool {
	return bizhours.MatchesWeeklyOff(targetWeekday, weeklyOffStr)
}

// CalculateStoreLiveStatus evaluates the real-time open/close state of a shop against current Indian Standard Time (IST).
func CalculateStoreLiveStatus(isOpenManual bool, openTimeStr, closeTimeStr, weeklyOffStr string) LiveTimingInfo {
	return CalculateStoreLiveStatusAt(isOpenManual, openTimeStr, closeTimeStr, weeklyOffStr, IndianNow())
}

// CalculateStoreLiveStatusAt evaluates open/close status against a given target time (critical for deterministic unit testing).
func CalculateStoreLiveStatusAt(isOpenManual bool, openTimeStr, closeTimeStr, weeklyOffStr string, targetTime time.Time) LiveTimingInfo {
	res := bizhours.CalculateAt(isOpenManual, openTimeStr, closeTimeStr, weeklyOffStr, targetTime)

	// Preserve exact customer banner text for emergency owner closure
	statusMsg := res.StatusMessage
	if !isOpenManual {
		statusMsg = "Temporarily closed by store owner"
	}

	return LiveTimingInfo{
		Status:         ShopLiveStatus(res.Status),
		IsOpenNow:      res.IsOpenNow,
		StatusMessage:  statusMsg,
		NextChangeAt:   res.NextChangeAt,
		MinutesToClose: res.MinutesToClose,
	}
}
