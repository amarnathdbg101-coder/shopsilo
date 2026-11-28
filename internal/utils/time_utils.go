package utils

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ctxKeyTimezone struct{}

var IndianLocation *time.Location

func init() {
	var err error
	IndianLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		IndianLocation = time.FixedZone("IST", 5*3600+1800)
	}
}

// ResolveLocation resolves a *time.Location from client-provided IANA timezone name
// (e.g. "Asia/Kolkata") or UTC offset in minutes (e.g. "330"), defaulting to IndianLocation.
func ResolveLocation(tzName, offsetMinutesStr string) *time.Location {
	cleanName := strings.TrimSpace(tzName)
	if cleanName != "" {
		if loc, err := time.LoadLocation(cleanName); err == nil {
			return loc
		}
	}

	if cleanOff := strings.TrimSpace(offsetMinutesStr); cleanOff != "" {
		if mins, err := strconv.Atoi(cleanOff); err == nil && mins >= -720 && mins <= 840 {
			label := cleanName
			if label == "" {
				sign := "+"
				abs := mins
				if abs < 0 {
					sign = "-"
					abs = -abs
				}
				label = fmt.Sprintf("UTC%s%02d:%02d", sign, abs/60, abs%60)
			}
			return time.FixedZone(label, mins*60)
		}
	}

	return IndianLocation
}

// WithLocation attaches the user's resolved *time.Location to ctx.
func WithLocation(ctx context.Context, loc *time.Location) context.Context {
	if loc == nil {
		loc = IndianLocation
	}
	return context.WithValue(ctx, ctxKeyTimezone{}, loc)
}

// LocationFromContext returns the user's device *time.Location from ctx, or IndianLocation as fallback.
func LocationFromContext(ctx context.Context) *time.Location {
	if ctx != nil {
		if loc, ok := ctx.Value(ctxKeyTimezone{}).(*time.Location); ok && loc != nil {
			return loc
		}
	}
	return IndianLocation
}

// NowInContext returns the current time in the user's device timezone from ctx (fallback IST).
func NowInContext(ctx context.Context) time.Time {
	return time.Now().In(LocationFromContext(ctx))
}

// InUserZone converts t into the user's device timezone from ctx (fallback IST).
func InUserZone(ctx context.Context, t time.Time) time.Time {
	return t.In(LocationFromContext(ctx))
}

// IndianNow returns current time in Indian Standard Time (Asia/Kolkata, UTC+5:30)
func IndianNow() time.Time {
	return time.Now().In(IndianLocation)
}

// IndianDayBounds computes start (00:00:00 IST) and end (24:00:00 IST) of the target date,
// returned as UTC time.Time for querying PostgreSQL TIMESTAMP/TIMESTAMPTZ columns.
func IndianDayBounds(t time.Time) (startUTC time.Time, endUTC time.Time) {
	inIST := t.In(IndianLocation)
	startIST := time.Date(inIST.Year(), inIST.Month(), inIST.Day(), 0, 0, 0, 0, IndianLocation)
	endIST := startIST.Add(24 * time.Hour)
	return startIST.UTC(), endIST.UTC()
}
