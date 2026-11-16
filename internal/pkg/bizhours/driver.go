package bizhours

// Status represents the evaluated availability state.
type Status string

const (
	StatusOpen        Status = "OPEN"
	StatusClosingSoon Status = "CLOSING_SOON"
	StatusClosed      Status = "CLOSED"
	StatusWeeklyOff   Status = "WEEKLY_OFF"
)

// LiveStatus contains user-facing banner strings, countdowns, and operational flags.
type LiveStatus struct {
	Status         Status `json:"status"`           // OPEN, CLOSING_SOON, CLOSED, WEEKLY_OFF
	IsOpenNow      bool   `json:"is_open_now"`       // True if business is actively operating
	StatusMessage  string `json:"status_message"`   // e.g. "Open now • Closes at 9:00 PM"
	NextChangeAt   string `json:"next_change_at"`   // e.g. "Closes in 25 mins" or "Opens tomorrow at 9:00 AM"
	MinutesToClose int    `json:"minutes_to_close"` // Remaining operating minutes (when OPEN or CLOSING_SOON)
}

// DaySchedule defines operating hours for an individual day.
type DaySchedule struct {
	IsOpen    bool   `json:"is_open"`
	OpenTime  string `json:"open_time"`  // "HH:MM" e.g. "09:00"
	CloseTime string `json:"close_time"` // "HH:MM" e.g. "21:30"
}
