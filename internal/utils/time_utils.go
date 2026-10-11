package utils

import "time"

var IndianLocation *time.Location

func init() {
	var err error
	IndianLocation, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		IndianLocation = time.FixedZone("IST", 5*3600+1800)
	}
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
