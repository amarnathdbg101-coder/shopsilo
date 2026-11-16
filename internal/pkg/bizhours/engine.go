package bizhours

import (
	"fmt"
	"time"
)

// Engine evaluates business availability and countdown telemetry.
type Engine struct {
	cfg *Config
}

// NewEngine creates an operating hours Engine.
func NewEngine(opts ...Option) *Engine {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Engine{cfg: cfg}
}

// Evaluate calculates status against current time.
func (e *Engine) Evaluate(isOpenManual bool, openTime, closeTime, weeklyOff string) LiveStatus {
	return e.EvaluateAt(isOpenManual, openTime, closeTime, weeklyOff, time.Now().In(e.cfg.Location))
}

// EvaluateAt calculates status against a specific reference time (essential for deterministic testing).
func (e *Engine) EvaluateAt(isOpenManual bool, openTimeStr, closeTimeStr, weeklyOffStr string, target time.Time) LiveStatus {
	openFormatted := Format12Hour(openTimeStr)
	closeFormatted := Format12Hour(closeTimeStr)
	if openFormatted == "" {
		openFormatted = "9:00 AM"
	}
	if closeFormatted == "" {
		closeFormatted = "9:00 PM"
	}

	// 1. Weekly Off / Holiday check
	if MatchesWeeklyOff(target.Weekday(), weeklyOffStr) {
		return LiveStatus{
			Status:         StatusWeeklyOff,
			IsOpenNow:      false,
			StatusMessage:  "Closed • Weekly holiday today",
			NextChangeAt:   fmt.Sprintf("Opens tomorrow at %s", openFormatted),
			MinutesToClose: 0,
		}
	}

	// 2. Manual switch check
	if !isOpenManual {
		return LiveStatus{
			Status:         StatusClosed,
			IsOpenNow:      false,
			StatusMessage:  "Temporarily closed by business owner",
			NextChangeAt:   fmt.Sprintf("Opens tomorrow at %s", openFormatted),
			MinutesToClose: 0,
		}
	}

	// 3. Parse minutes from midnight
	openMin := ParseMinutesFromMidnight(openTimeStr, 9*60)
	closeMin := ParseMinutesFromMidnight(closeTimeStr, 21*60)
	currentMin := target.Hour()*60 + target.Minute()

	// 24-Hour continuous operation check
	if (openMin == 0 && closeMin == 0) || (openMin == 0 && closeMin >= 1439) {
		return LiveStatus{
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
		// Normal daytime operating shift (e.g. 09:00 to 21:00)
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
		if minutesToClose <= e.cfg.ClosingSoonThreshold && minutesToClose > 0 {
			return LiveStatus{
				Status:         StatusClosingSoon,
				IsOpenNow:      true,
				StatusMessage:  fmt.Sprintf("Closing soon • %d mins left", minutesToClose),
				NextChangeAt:   fmt.Sprintf("Closes at %s", closeFormatted),
				MinutesToClose: minutesToClose,
			}
		}
		return LiveStatus{
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
		// In overnight shift, before shift begins
		nextOpenMsg = fmt.Sprintf("Opens today at %s", openFormatted)
	}

	return LiveStatus{
		Status:         StatusClosed,
		IsOpenNow:      false,
		StatusMessage:  fmt.Sprintf("Closed • %s", nextOpenMsg),
		NextChangeAt:   nextOpenMsg,
		MinutesToClose: 0,
	}
}
