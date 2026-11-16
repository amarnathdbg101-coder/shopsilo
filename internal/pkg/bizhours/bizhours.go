// Package bizhours evaluates business availability, daytime and overnight shifts,
// weekly off days, and real-time open/closed customer banners with zero external dependencies.
package bizhours

import "time"

var defaultEngine = NewEngine()

// Calculate evaluates availability against current system time.
func Calculate(isOpenManual bool, openTime, closeTime, weeklyOff string) LiveStatus {
	return defaultEngine.Evaluate(isOpenManual, openTime, closeTime, weeklyOff)
}

// CalculateAt evaluates availability at a specific target time.
func CalculateAt(isOpenManual bool, openTime, closeTime, weeklyOff string, target time.Time) LiveStatus {
	return defaultEngine.EvaluateAt(isOpenManual, openTime, closeTime, weeklyOff, target)
}

// FormatTime converts a 24-hr "HH:MM" string to a user-friendly 12-hr representation (e.g. "21:30" -> "9:30 PM").
func FormatTime(timeStr string) string {
	return Format12Hour(timeStr)
}
