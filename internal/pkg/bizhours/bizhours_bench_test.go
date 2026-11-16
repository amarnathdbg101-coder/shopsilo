package bizhours

import (
	"testing"
	"time"
)

func BenchmarkBizHours_CalculateAt(b *testing.B) {
	target := time.Date(2026, 11, 5, 14, 30, 0, 0, time.UTC)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = CalculateAt(true, "09:00", "21:00", "Sunday", target)
	}
}

func BenchmarkBizHours_FormatTime(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = FormatTime("21:30")
	}
}

func BenchmarkBizHours_ParseMinutes(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ParseMinutesFromMidnight("21:30", 0)
	}
}
